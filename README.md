## Syncloud (https://syncloud.org)

Simple self-hosting of cloud apps.

It is available as an image or a pre-installed device.

We are open to cooperation with hardware vendors interested in including Syncloud into their products.

### Apps

* [Nextcloud](https://nextcloud.com/): File sharing, calendar, contacts.
* [Diaspora](https://diasporafoundation.org/): Social network.
* [Rocketchat](https://rocket.chat/): Text, voice and video messaging.
* [Mail](https://roundcube.net/): Email messaging with Roundcube web.
* [GOGS](https://gogs.io/): Git source code hosting.
* [Syncthing](https://syncthing.net/): File synchronization between devices.
* [WordPress](https://wordpress.org/): Blogging, mailing lists and forums, media galleries, and online stores.
* [Notes](https://standardnotes.org/): Safe place for your notes, thoughts, and life's work.

### Images

https://github.com/syncloud/platform/wiki

## For developers

Syncloud platform manages the installation and device settings.

### Web UI development

install NodeJS

````
cd www/public
npm install
npm start
````
### Running local drone build

Get drone cli binary: http://docs.drone.io/cli-installation/
````
sudo DOCKER_API_VERSION=1.24 arch=[amd64|arm] /path/to/cli/drone exec
````

### Build server

http://build.syncloud.org/syncloud/platform

### Build artifacts (screenshots, system logs)

http://artifact.syncloud.org/platform
