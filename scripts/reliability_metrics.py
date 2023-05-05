#!/usr/bin/env python3.9
import os
import logging
import threading
import argparse
from builtins import staticmethod
import subprocess

from ottopia_logging.logging_factory import LoggingFactory
from ottopia_logging.log_level import LogLevel
from ottopia_logging.logging_settings import LoggingSettings


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
        self.name: str = state
        self._log: logging.Logger = Utils.get_logger(f'KeepalivedState{self.name}')
        self._logstash: logging.LoggerAdapter = Utils.get_logstash_logger(f'KeepalivedState{self.name}')

    def report(self):
        self._logstash.info(f'Entering {self.name} state')


class MasterState(StateBase):
    def __init__(self):
        super().__init__('Master')


class BackupState(StateBase):
    def __init__(self):
        super().__init__('Backup')


class Unit:
    def __init__(self, name: str):
        self.name: str = name
        self.thread: threading.Thread = threading.Thread(group=None, name=self.name, target=self.analyze)
        self._log: logging.Logger = Utils.get_logger(self.name)
        self._logstash: logging.LoggerAdapter = Utils.get_logstash_logger(self.name)

    def start(self):
        self._log.debug(f'starting unit {self.name}')
        self.thread.start()

    def stop(self):
        self._log.debug(f'stopping unit {self.name}')
        self.thread.join()

    def analyze(self):
        self._log.debug(f'analyzing unit {self.name}')


class Service(Unit):
    def __init__(self, name: str):
        super().__init__(name)

    def analyze(self):
        super().analyze()
        try:
            process: subprocess.CompletedProcess = subprocess.run(['systemctl', 'is-active', self.name], check=True)
            self._log.info(f'service {self.name} is active')
        except subprocess.CalledProcessError as e:
            self._logstash.error(f'service {self.name} is not active. Error: {e}')
        except Exception as e:
            self._log.critical(e)


class Container(Unit):
    def __init__(self, name: str):
        super().__init__(name)

    def analyze(self):
        super().analyze()
        failed: bool = True
        status: str = 'unknown (not running and not found)' # default value
        try:
            process: subprocess.CompletedProcess = subprocess.run(['docker', 'ps', '-a', '--filter', f'name={self.name}'], capture_output=True, check=True)
            if process.stdout:
                stdout_lines = process.stdout.splitlines()
                if len(stdout_lines) > 1:
                    pos_status: int = stdout_lines[0].find(b'STATUS')
                    if pos_status != -1:
                        pos_ports: int = stdout_lines[0].find(b'PORTS')
                        if pos_ports != -1:
                            status: str = stdout_lines[1][pos_status:pos_ports].strip()
                            if b'Up' in status:
                                failed = False
        except subprocess.CalledProcessError as e:
            self._log.error(e)
        except Exception as e:
            self._log.critical(e)
        finally:
            if failed:
                self._logstash.info(f'container {self.name} failed. Status: {status}')


class FaultState(StateBase):
    SERVICE_NAMES: list[str] = ['tca.service', 'relayserver.service', 'docker.service', 'docker.socket']
    CONTAINER_NAMES: list[str] = ['assistance-session-manager', 'connection-manager', 'station-manager']

    def __init__(self):
        super().__init__('Fault')
        self.services: list[Service] = [Service(service_name) for service_name in FaultState.SERVICE_NAMES]
        self.containers: list[Container] = [Container(f'tca-{container_name}-1') for container_name in FaultState.CONTAINER_NAMES]
        self.units: list[Unit] = self.services + self.containers

    def report(self):
        super().report()

        '''
        search for unit errors
        '''

        for unit in self.units:
            unit.start()
        for unit in self.units:
            unit.stop()


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


class Utils:
    args: argparse.Namespace = None
    handler: logging.Handler = None
    loggers: dict = {}
    logger_adapters: dict = {}
    @staticmethod
    def get_arguments() -> argparse.Namespace:
        if Utils.args is not None:
            return Utils.args
        parser: argparse.ArgumentParser = argparse.ArgumentParser(description='find services errors in syslog')
        parser.add_argument('state', choices=['MASTER', 'BACKUP', 'FAULT'], help='state of the machine')
        Utils.args = parser.parse_args()
        return Utils.args

    @staticmethod
    def get_file_handler() -> logging.Handler:
        if Utils.handler is not None:
            return Utils.handler

        file_name: str = f'/tmp/{os.path.basename(__file__).split(".")[0]}.log'
        formatter: logging.Formatter = logging.Formatter('[%(asctime)s] [%(levelname)s] [%(name)s] [%(thread)d] [%(funcName)s:%(lineno)d] %(message)s')
        Utils.handler: logging.Handler = logging.handlers.RotatingFileHandler(file_name, maxBytes=0x10000,
                                      backupCount=10)
        Utils.handler.setFormatter(formatter)
        return Utils.handler

    @staticmethod
    def get_logger(name: str) -> logging.Logger:
        if name in Utils.loggers:
            return Utils.loggers[name]
        logger: logging.Logger = logging.getLogger(name)
        handler: logging.Handler = Utils.get_file_handler()
        logger.setLevel(logging.INFO)
        logger.handlers.clear()
        logger.addHandler(handler)
        Utils.loggers[name] = logger
        return logger

    @staticmethod
    def get_logstash_logger(name: str) -> logging.LoggerAdapter:
        if name in Utils.logger_adapters:
            return Utils.logger_adapters[name]
        logger_adapter: logging.LoggerAdapter = LoggingFactory.get_logger(
            module_name=name,
            logging_settings=LoggerSetting(),
        )
        handler: logging.Handler = Utils.get_file_handler()
        logger_adapter.logger.addHandler(handler)
        Utils.logger_adapters[name] = logger_adapter
        return logger_adapter


def main() -> int:
    logger: logging.Logger = Utils.get_logger(__name__)
    logger.info('starting')

    args: argparse.Namespace = Utils.get_arguments()

    StateFactory.init_states()
    state: StateBase = StateFactory.get_state(args.state.upper())
    state.report()

    logger.info('done')
    return 0


#import unittest
# write unit tests for each function in this module
#def test_search_for_services_errors():
#    pass


if __name__ == "__main__":
    exit(main())
