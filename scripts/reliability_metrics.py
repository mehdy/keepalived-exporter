#!/usr/bin/env python3
import datetime
import os
import logging
import threading
import queue
import re

#from ottopia_logging.logging_factory import LoggingFactory
#from settings import settings

#logger = LoggingFactory.get_logger(
#    module_name=__name__,
#    logging_settings=settings.logging_settings,
#)


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
                logger = logging.getLogger(f'{self.name}')
                logger.info(f'found on {line_data.syslog_date}: {fail_line}')
                # TODO: send to logstash

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

    def put(self, line_data: LineData):
        logger = logging.getLogger(f'{self.name}')
        logger.debug('putting line on queue')
        self.queue.put(line_data)


def put_chunk_lines_on_queue(chunk: str, services: list[Service], current_time: datetime.datetime, regex: re.Pattern[str]) -> bool:
    '''
    put lines from chunk on queue, return True if chunk is in range, False otherwise
    '''

    logger = logging.getLogger(__name__)
    THRESHOLD_SECONDS: datetime.timedelta = datetime.timedelta(seconds=1000000)
    lines: list[str] = reversed(chunk.split('\n'))
    continue_flag: bool = True
    for line in lines:
        date_str: str = line[: 15]
        try:
            if not regex.match(date_str):
                continue

            syslog_date: datetime.datetime = datetime.datetime.strptime(date_str, '%b %d %H:%M:%S').replace(year=current_time.year)
            continue_flag &= syslog_date + THRESHOLD_SECONDS >= current_time


            if continue_flag:
                line_data: LineData = LineData(line[17:], syslog_date)
                for service in services:
                    service.put(line_data)
            else:
                logger.debug(f'{line} not in range, {current_time}')
                break

        except ValueError:
            continue_flag = False

    return continue_flag


def search_syslog_for_errors_in_services(services: list[Service]) -> int:
    '''
    open syslog file
    search for services errors from the back of the file
    and don't look past the last reboot and the last 10 seconds
    return 0 if no errors found
    return 1 if errors found
    return 2 if syslog file not found
    '''
    current_time: datetime.datetime = datetime.datetime.now()
    if not os.path.exists("/var/log/syslog"):
        return 2

    # date format is: 'MMM dd HH:MM:SS'
    regex: re.Pattern[str] = re.compile(r'(\w{3} \d{2} \d{2}:\d{2}:\d{2})')
    MAX_CHUNK_SIZE = 0x1000
    try:
        prev_chunk_remainder: str = ''
        with open("/var/log/syslog", "r") as syslog:
            syslog.seek(0, os.SEEK_END) #go to end of file
            continue_flag: bool = True
            while continue_flag and syslog.tell() > 0:
                chunk_size = min(syslog.tell(), MAX_CHUNK_SIZE)
                interval: int = syslog.tell() - chunk_size
                syslog.seek(interval, os.SEEK_SET)
                chunk: str = syslog.read(chunk_size) + prev_chunk_remainder
                syslog.seek(syslog.tell() - chunk_size, os.SEEK_SET)

                pos = chunk.find('\n')
                prev_chunk_remainder = chunk[:pos]

                continue_flag = put_chunk_lines_on_queue(chunk[pos + 1:], services, current_time, regex)

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

    SERVICE_NAMES = ('tca', 'relayserver')
    event: threading.Event = threading.Event()
    services: list[Service] = []
    for service_name in SERVICE_NAMES:
        service: Service = Service(service_name, event)
        service.start()
        services.append(service)
    ret_val: int = search_syslog_for_errors_in_services(services)

    event.set()
    for service in services:
        service.stop()

    logger = logging.getLogger(__name__)
    logger.info('done')
    return ret_val


import unittest
# write unit tests for each function in this module
def test_search_syslog_for_errors_in_services():
    pass


if __name__ == "__main__":
    exit(main())
