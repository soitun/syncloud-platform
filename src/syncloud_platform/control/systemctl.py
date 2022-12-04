import os
from os.path import join
import shutil
from string import Template
import string
from subprocess import check_output, CalledProcessError
from syncloudlib import logger

SYSTEMD_DIR = join('/lib', 'systemd', 'system')


class Systemctl:

    def __init__(self, platform_config):
        self.platform_config = platform_config
        self.log = logger.get_logger('systemctl')

    def __remove(self, filename):

        if self.__stop(filename) in ("unknown", "inactive"):
            return
        try:
            check_output('systemctl disable {0} 2>&1'.format(filename), shell=True)
        except CalledProcessError as e:
            self.log.error(e.output.decode())
            raise e
        systemd_file = self.__systemd_file(filename)
        if os.path.isfile(systemd_file):
            os.remove(systemd_file)

    def restart_service(self, service):

        self.stop_service(service)
        self.start_service(service)

    def start_service(self, service):
        service = self.service_name(service)
        self.__start('{0}.service'.format(service))

    def __start(self, service):
        log = logger.get_logger('systemctl')

        try:
            log.info('starting {0}'.format(service))
            check_output('systemctl start {0} 2>&1'.format(service), shell=True)
        except CalledProcessError as error:
            log.error(error.output.decode())
            try:
                log.error(check_output('journalctl -u {0} 2>&1'.format(service), shell=True).decode())
            except CalledProcessError as e:
                log.error(e.output.decode())
            raise error

    def stop_service(self, service):
        service = self.service_name(service)
        return self.__stop('{0}.service'.format(service))

    def __stop(self, service):
        log = logger.get_logger('systemctl')

        try:
            log.info('checking {0}'.format(service))
            # TODO: exit code 3 when inactive
            result = check_output('systemctl is-active {0} 2>&1'.format(service), shell=True).decode().strip()
            log.info('stopping {0}'.format(service))
            check_output('systemctl stop {0} 2>&1'.format(service), shell=True)
        except CalledProcessError as e:
            result = e.output.decode().strip()

        log.info("{0}: {1}".format(service, result))
        return result

    def service_name(self, service):
        return "snap.{0}".format(service)

    def __systemd_file(self, filename):
        return join(SYSTEMD_DIR, filename)


