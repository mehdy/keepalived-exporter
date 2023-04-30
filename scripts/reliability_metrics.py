#!/usr/bin/env python3.9
import datetime
import os
import logging
import threading
import queue
import re
from ottopia_logging.logging_factory import LoggingFactory
from ottopia_logging.log_level import LogLevel
from ottopia_logging.logging_settings import LoggingSettings


class LoggerSetting(LoggingSettings):
    """
    Logger settings
    """

    DEFAULT_COMPONENT_NAME: str = "keepalived-exporter"

    DEFAULT_LOG_LEVEL: LogLevel = LogLevel.INFO

    class Config:
        """
        Config for Logger settings
        """

        arbitrary_types_allowed = True



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

    def find(self):
        fail_line: str = f'{self.name}.service: Main process exited, code=exited, status=1/FAILURE'
        while not self.event.is_set() or not self.queue.empty():
            line_data: LineData = self.queue.get()
            if line_data and fail_line in line_data.line:
                self.report_to_logstash(line_data.syslog_date, fail_line)

    def report_to_logstash(self, syslog_date: datetime.datetime, fail_line: str):
        logstash_logger = LoggingFactory.get_logger(
            module_name=self.name,
            logging_settings=LoggerSetting(),
        )
        logstash_logger.info(f'service {self.name} failed on {syslog_date}: {fail_line}')

    def start(self):
        logger = logging.getLogger(f'{self.name}')
        logger.info('starting service')
        self.thread.start()

    def stop(self):
        logger = logging.getLogger(f'{self.name}')
        logger.info('stopping service')

        if self.queue.empty() and self.thread.is_alive():
            logger.debug('putting None on queue')
            self.queue.put(None)
        self.thread.join()

    def async_analyze(self, line_data: LineData):
        logger = logging.getLogger(f'{self.name}')
        logger.debug('putting line on queue')
        self.queue.put(line_data)


class SyslogParser:
    SYSLOG_PATH: str = '/var/log/syslog'
    def __init__(self, services: list[Service]):
        self.services: list[Service] = services
        self.current_time: datetime.datetime = datetime.datetime.now()
        # date format is: 'MMM dd HH:MM:SS'
        self.regex: re.Pattern[str] = re.compile(r'(\w{3} \d{2} \d{2}:\d{2}:\d{2})')



    def put_chunk_lines_on_queue(self, chunk: str) -> bool:
        '''
        put lines from chunk on queue, return True if chunk is in range, False otherwise
        '''

        logger = logging.getLogger(__name__)
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
                    logger.debug(f'{line} not in range, {self.current_time}')
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
            logger = logging.getLogger(__name__)
            logger.exception(e)
            return 1

        return 0


def init_logging():
    logging.basicConfig(filename='example.log', format='[%(asctime)s] [%(levelname)s] [%(name)s] [%(funcName)s:%(lineno)d] %(message)s', filemode='w', level=logging.INFO)
    logger = logging.getLogger(__name__)
    logger.addHandler(logging.StreamHandler())
    logger.info('starting')


def main() -> int:
    init_logging()

    event: threading.Event = threading.Event()
    SERVICE_NAMES: list[str] = ['tca', 'relayserver']
    services: list[Service] = [Service(service_name, event) for service_name in SERVICE_NAMES]
    for service in services:
        service.start()

    syslog_parser: SyslogParser = SyslogParser(services)
    ret_val: int = syslog_parser.search_for_services_errors()

    event.set()
    for service in services:
        service.stop()

    logger = logging.getLogger(__name__)
    logger.info('done')
    return ret_val


import unittest
# write unit tests for each function in this module
def test_search_for_services_errors():
    pass


if __name__ == "__main__":
    exit(main())

