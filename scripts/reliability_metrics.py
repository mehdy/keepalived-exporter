#!/usr/bin/env python3.9
import datetime
import os
import logging
import threading
import queue
import re
import argparse
from builtins import staticmethod
import docker
import docker.models.containers

from ottopia_logging.logging_factory import LoggingFactory
from ottopia_logging.log_level import LogLevel
from ottopia_logging.logging_settings import LoggingSettings
import dataclasses


class LoggerSetting(LoggingSettings):
    """
    Logger settings
    """

    DEFAULT_COMPONENT_NAME: str = "reliability-metrics"

    DEFAULT_LOG_LEVEL: LogLevel = LogLevel.INFO

    class Config:
        """
        Config for Logger settings
        """

        arbitrary_types_allowed = True


class StateBase:
    def __init__(self, state: str):
        self._name: str = f'KeepalivedState{state}'
        self._log: logging.Logger = get_logger(self._name)
        self._logstash = LoggingFactory.get_logger(
            module_name=self._name,
            logging_settings=LoggerSetting(),
        )
        self.current_state: str = state

    def report(self) -> int:
        self._logstash.info(f'Entering {self.current_state} state')
        return 0


class MasterState(StateBase):
    def __init__(self):
        super().__init__('Master')


class BackupState(StateBase):
    def __init__(self):
        super().__init__('Backup')


@dataclasses.dataclass
class LineData:
    def __init__(self, line: str, syslog_date: datetime.datetime):
        self.line = line
        self.syslog_date = syslog_date


class Service:
    def __init__(self, name: str, event: threading.Event):
        self.name = name
        self.event = event
        self.queue = queue.Queue()
        self.thread = threading.Thread(group=None, target=self.find)
        self._log = get_logger(f'{self.name}')
        self._logstash = LoggingFactory.get_logger(
            module_name=self.name,
            logging_settings=LoggerSetting(),
        )

    def find(self):
        fail_line: str = f'{self.name}.service: Main process exited, code=exited, status=1/FAILURE'
        while not self.event.is_set() or not self.queue.empty():
            self._log.debug('getting line from queue')
            try:
                line_data: LineData = self.queue.get(timeout=0.1)
                self._log.debug(f'got line from queue, queue size:: {self.queue.qsize()}')
            except queue.Empty:
                continue
            if line_data and fail_line in line_data.line:
                self.report_to_logstash(line_data.syslog_date, fail_line)

    def report_to_logstash(self, syslog_date: datetime.datetime, fail_line: str):
        self._logstash.info(f'service {self.name} failed on {syslog_date}: {fail_line}')

    def start(self):
        self._log.info('starting service')
        self.thread.start()

    def stop(self):
        self._log.info('stopping service')

        if self.queue.empty() and self.thread.is_alive():
            self._log.debug('putting None on queue')
            self.queue.put(None)
        self.thread.join()

    def async_analyze(self, line_data: LineData):
        self._log.debug('putting line on queue')
        self.queue.put(line_data)


class Container:
    def __init__(self, name: str):
        self.name = name
        self.container: docker.models.containers.Container = None
        self.thread = threading.Thread(group=None, target=self.async_analyze)
        self._log = get_logger(f'{self.name}')
        self._logstash = LoggingFactory.get_logger(
            module_name=self.name,
            logging_settings=LoggerSetting(),
        )

    def report_to_logstash(self, status: str):
        self._logstash.info(f'container {self.name} is not running. Status: {status}')

    def start(self):
        self._log.info('starting container')
        self.thread.start()

    def stop(self):
        self._log.info('stopping container')
        self.thread.join()

    def async_analyze(self):
        client: docker.client.DockerClient = docker.from_env()
        self.container: docker.models.containers.Container = client.containers.get(self.name)
        self._log.debug(f'self.container is none: {self.container is None}')
        if self.container and self.container.status != 'running':
            self.report_to_logstash(self.container.status)


class FaultState(StateBase):
    SYSLOG_PATH: str = '/var/log/syslog'
    SERVICE_NAMES: list[str] = ['tca', 'relayserver']
    CONTAINER_NAMES: list[str] = ['assistance-session-manager', 'connection-manager', 'station-manager']

    def __init__(self):
        super().__init__('Fault')
        self.current_time: datetime.datetime = datetime.datetime.now()
        # date format is: 'MMM dd HH:MM:SS'
        self.regex: re.Pattern[str] = re.compile(r'(\w{3} \d{2} \d{2}:\d{2}:\d{2})')

        self.event: threading.Event = threading.Event()
        self.services: list[Service] = [Service(service_name, self.event) for service_name in FaultState.SERVICE_NAMES]
        self.containers: list[Container] = [Container(f'tca-{container_name}-1') for container_name in FaultState.CONTAINER_NAMES]

    def put_chunk_lines_on_queue(self, chunk: str) -> bool:
        '''
        put lines from chunk on queue, return True if chunk is in range, False otherwise
        '''

        THRESHOLD_SECONDS: datetime.timedelta = datetime.timedelta(seconds=5)
        lines: reversed[str] = reversed(chunk.split('\n'))
        continue_flag: bool = True
        for line in lines:
            date_str: str = line[: 15]
            try:
                if not self.regex.match(date_str):
                    continue

                # date format is: 'MMM dd HH:MM:SS'
                syslog_date: datetime.datetime = datetime.datetime.strptime(date_str, '%b %d %H:%M:%S').replace(year=self.current_time.year)
                continue_flag &= syslog_date + THRESHOLD_SECONDS >= self.current_time


                if continue_flag:
                    line_data: LineData = LineData(line[17:], syslog_date)
                    for service in self.services:
                        service.async_analyze(line_data)
                else:
                    self._log.debug(f'{line} not in range, {self.current_time}')
                    break

            except ValueError:
                continue_flag = False

        return continue_flag

    def search_for_services_errors(self) -> int:
        '''
        open syslog file
        search for services errors from the back of the file
        and don't look past the last reboot and the last 10 seconds
        return 0 if no errors found
        return 1 if errors found
        return 2 if syslog file not found
        '''
        if not os.path.exists(self.SYSLOG_PATH):
            return 2

        for service in self.services:
            service.start()

        MAX_CHUNK_SIZE = 0x1000
        try:
            prev_chunk_remainder: str = ''
            with open(self.SYSLOG_PATH, 'r') as syslog:
                syslog.seek(0, os.SEEK_END) #go to end of file
                continue_flag: bool = True
                while continue_flag and syslog.tell() > 0:
                    chunk_size = min(syslog.tell(), MAX_CHUNK_SIZE)
                    syslog.seek(syslog.tell() - chunk_size, os.SEEK_SET)
                    chunk: str = syslog.read(chunk_size) + prev_chunk_remainder
                    syslog.seek(syslog.tell() - chunk_size, os.SEEK_SET)

                    pos = chunk.find('\n')
                    prev_chunk_remainder = chunk[:pos]

                    continue_flag = self.put_chunk_lines_on_queue(chunk[pos + 1:])

        except Exception as e:
            self._log.exception(e)
            return 1

        finally:
            self._log.debug('stopping services')
            self.event.set()
            for service in self.services:
                service.stop()

        return 0

    def search_for_containers_errors(self):
        '''
        search for containers errors
        '''

        for container in self.containers:
            container.start()
        for container in self.containers:
            container.stop()

    def report(self) -> int:
        super().report()

        ret_val: int = self.search_for_services_errors()
        self.search_for_containers_errors()

        return ret_val


class StateFactory:
    states: dict = {}

    @staticmethod
    def get_state(state: str) -> StateBase:
        if state in StateFactory.states:
            return StateFactory.states[state]()
        else:
            raise ValueError(f'Invalid state: {state}. State must be one of {StateFactory.states.keys()}')

    @staticmethod
    def register_state(state: str, state_class: StateBase):
        StateFactory.states[state] = state_class

    @staticmethod
    def init_states():
        StateFactory.register_state('MASTER', MasterState)
        StateFactory.register_state('BACKUP', BackupState)
        StateFactory.register_state('FAULT', FaultState)



def get_arguments() -> argparse.Namespace:
    parser: argparse.ArgumentParser = argparse.ArgumentParser(description='find services errors in syslog')
    parser.add_argument('state', choices=['MASTER', 'BACKUP', 'FAULT'], help='state of the machine')
    return parser.parse_args()


def get_logger(name: str) -> logging.Logger:
    log_file_name: str = os.path.basename(__file__).split('.')[0] + '.log'
    formatter: logging.Formatter = logging.Formatter('[%(asctime)s] [%(levelname)s] [%(name)s] [%(funcName)s:%(lineno)d] %(message)s')
    handler: logging.Handler = logging.handlers.RotatingFileHandler(log_file_name, maxBytes=0x10000,
                                  backupCount=10)
    handler.setFormatter(formatter)
    logger = logging.getLogger(name)
    logger.addHandler(handler)

    return logger


def main() -> int:
    logger = get_logger(__name__)
    logger.info('starting')

    args: argparse.Namespace = get_arguments()

    StateFactory.init_states()
    state: StateBase = StateFactory.get_state(args.state.upper())
    ret_val: int = state.report()

    logger.info('done')
    return ret_val


#import unittest
# write unit tests for each function in this module
#def test_search_for_services_errors():
#    pass


if __name__ == "__main__":
    exit(main())

