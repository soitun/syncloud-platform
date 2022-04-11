local name = "platform";
local browser = "chrome";
local go = "1.17.3";

local build(arch, testUI) = [{
    kind: "pipeline",
    name: arch,

    platform: {
        os: "linux",
        arch: arch
    },
    steps: [
        {
            name: "version",
            image: "debian:buster-slim",
            commands: [
                "echo $DRONE_BUILD_NUMBER > version",
                "echo " + arch + "-$DRONE_BRANCH > domain"
            ]
        },
        {
            name: "build web",
            image: "node:16.1.0-alpine3.12",
            commands: [
                "apk add --update --no-cache python2 alpine-sdk ",
                "mkdir -p build/platform",
                "cd www",
                "npm install",
                "npm run test",
                "npm run lint",
                "npm run build",
                "cp -r dist ../build/platform/www"
            ]
        },
        {
            name: "build backend",
            image: "golang:" + go,
            commands: [
                "cd backend",
                "go test ./... -coverprofile cover.out",
                "go tool cover -func cover.out",
                "go build -ldflags '-linkmode external -extldflags -static' -o ../build/platform/bin/backend cmd/backend/main.go",
                "../build/platform/bin/backend -h",
                "go build -ldflags '-linkmode external -extldflags -static' -o ../build/platform/bin/cli cmd/cli/main.go",
                "../build/platform/bin/cli -h"
            ]
        },
        {
            name: "build api test",
            image: "golang:" + go,
            commands: [
                "cd integration/api",
                "go test -c -o api.test"
            ]
        },
        {
            name: "build uwsgi",
            image: "debian:buster-slim",
            commands: [
                "./build-uwsgi.sh"
            ],
            volumes: [
                {
                    name: "docker",
                    path: "/usr/bin/docker"
                },
                {
                    name: "docker.sock",
                    path: "/var/run/docker.sock"
                }
            ]
        },
        {
            name: "package",
            image: "debian:buster-slim",
            commands: [
                "VERSION=$(cat version)",
                "./build.sh $VERSION",
                "./integration/testapp/build.sh "
            ]
        },
        {
            name: "test-unit",
            image: "python:3.8-slim-buster",
            commands: [
              "apt update",
              "apt install -y build-essential libsasl2-dev libldap2-dev libssl-dev libjansson-dev libltdl7 libnss3 libffi-dev",
              "pip install -r requirements.txt",
              "pip install -r dev_requirements.txt",
              "cd src",
              "py.test test"
            ]
        }
    ] + ( if arch != "arm64" then [
        {
            name: "test-intergation-jessie",
            image: "python:3.8-slim-buster",
            environment: {
                REDIRECT_USER: {
                    from_secret: "REDIRECT_USER"
                },
                REDIRECT_PASSWORD: {
                    from_secret: "REDIRECT_PASSWORD"
                }
            },
            commands: [
              "apt-get update && apt-get install -y sshpass openssh-client netcat rustc apache2-utils libffi-dev",
              "./integration/wait-ssh.sh device-jessie",
              "pip install -r dev_requirements.txt",
              "cd integration",
              "py.test -x -s verify.py --distro=jessie --domain=$(cat ../domain) --app-archive-path=$(realpath ../*.snap) --device-host=device-jessie --app=" + name + " --arch=" + arch + " --redirect-user=$REDIRECT_USER --redirect-password=$REDIRECT_PASSWORD"
            ]
        }] else []) + [
  {
        name: "selenium-video",
        image: "selenium/video:ffmpeg-4.3.1-20220208",
        detach: true,
        environment: {
            "DISPLAY_CONTAINER_NAME": "selenium",
             FILE_NAME: "video.mkv"
        },
        volumes: [
            {
                name: "shm",
                path: "/dev/shm"
            },
           {
                name: "videos",
                path: "/videos"
            }
        ]
    },
          {
            name: "test-intergation-buster",
            image: "python:3.8-slim-buster",
            environment: {
                REDIRECT_USER: {
                    from_secret: "REDIRECT_USER"
                },
                REDIRECT_PASSWORD: {
                    from_secret: "REDIRECT_PASSWORD"
                }
            },
            commands: [
              "apt-get update && apt-get install -y sshpass openssh-client netcat rustc apache2-utils libffi-dev",
              "./integration/wait-ssh.sh device-buster",
              "pip install -r dev_requirements.txt",
              "cd integration",
              "py.test -x -s verify.py --distro=buster --domain=$(cat ../domain) --app-archive-path=$(realpath ../*.snap) --device-host=device-buster --app=" + name + " --arch=" + arch + " --redirect-user=$REDIRECT_USER --redirect-password=$REDIRECT_PASSWORD"
            ]
        }
    ] + ( if testUI then [
        {
            name: "test-ui-" + mode + "-" + distro,
            image: "python:3.8-slim-buster",
            environment: {
                REDIRECT_USER: {
                    from_secret: "REDIRECT_USER"
                },
                REDIRECT_PASSWORD: {
                    from_secret: "REDIRECT_PASSWORD"
                }
            },
            commands: [
              "apt-get update && apt-get install -y sshpass openssh-client libffi-dev",
              "pip install -r dev_requirements.txt",
              "cd integration",
              "py.test -x -s test-ui.py --distro=" + distro + " --ui-mode=" + mode + " --domain=$(cat ../domain) --device-host=device-" + distro + " --redirect-user=$REDIRECT_USER --redirect-password=$REDIRECT_PASSWORD --app=" + name + " --browser=" + browser
            ],
            volumes: [{
                name: "shm",
                path: "/dev/shm"
            }]
        } 
        for mode in ["desktop", "mobile"]
        for distro in ["buster", "jessie"] 
    ] else []) + [
        {
            name: "upload",
            image: "debian:buster-slim",
            environment: {
                AWS_ACCESS_KEY_ID: {
                    from_secret: "AWS_ACCESS_KEY_ID"
                },
                AWS_SECRET_ACCESS_KEY: {
                    from_secret: "AWS_SECRET_ACCESS_KEY"
                }
            },
            commands: [
              "PACKAGE=$(cat package.name)",
              "apt update && apt install -y wget",
              "wget https://github.com/syncloud/snapd/releases/download/1/syncloud-release-" + arch + " -O release --progress=dot:giga",
              "chmod +x release",
              "./release publish -f $PACKAGE -b $DRONE_BRANCH"
            ],
            when: {
                branch: ["stable", "master"]
            }
        },
        {
            name: "test-store",
            image: "python:3.8-slim-buster",
            
            commands: [
              "apt-get update && apt-get install -y sshpass openssh-client libffi-dev",
              "pip install -r dev_requirements.txt",
              "cd integration",
              "py.test -x -s test-store.py --distro=buster --domain=$(cat ../domain) --device-host=device-buster --app=" + name,
            ],
            when: {
                branch: ["stable", "master"]
            }
        },
        {
            name: "artifact",
            image: "appleboy/drone-scp",
            settings: {
                host: {
                    from_secret: "artifact_host"
                },
                username: "artifact",
                key: {
                    from_secret: "artifact_key"
                },
                timeout: "2m",
                command_timeout: "2m",
                target: "/home/artifact/repo/" + name + "/${DRONE_BUILD_NUMBER}-" + arch,
                source: "artifact/*",
                privileged: true,
		            strip_components: 1,
                volumes: [
                   {
                        name: "videos",
                        path: "/drone/src/artifact/videos"
                    }
               ]
           },
           when: {
              status: [ "failure", "success" ]
            }
        }
    ],
    trigger: {
      event: [
        "push",
        "pull_request"
      ]
    },
    services: ( if arch != "arm64" then [ 
        {
            name: "device-jessie",
            image: "syncloud/bootstrap-" + arch,
            privileged: true,
            volumes: [
                {
                    name: "dbus",
                    path: "/var/run/dbus"
                },
                {
                    name: "dev",
                    path: "/dev"
                }
            ]
        }] else []) + [
        {
            name: "device-buster",
            image: "syncloud/bootstrap-buster-" + arch,
            privileged: true,
            volumes: [
                {
                    name: "dbus",
                    path: "/var/run/dbus"
                },
                {
                    name: "dev",
                    path: "/dev"
                }
            ]
        }
    ] + ( if testUI then [{
            name: "selenium",
            image: "selenium/standalone-" + browser + ":4.1.2-20220208",
        environment: {
                SE_NODE_SESSION_TIMEOUT: "999999",
                START_XVFB: "true"
            },
               volumes: [{
                name: "shm",
                path: "/dev/shm"
            }]
        }
    ] else [] ),
    volumes: [
        {
            name: "dbus",
            host: {
                path: "/var/run/dbus"
            }
        },
        {
            name: "dev",
            host: {
                path: "/dev"
            }
        },
        {
            name: "shm",
            temp: {}
        },
        {
            name: "docker",
            host: {
                path: "/usr/bin/docker"
            }
        },
        {
            name: "docker.sock",
            host: {
                path: "/var/run/docker.sock"
            }
        },
      {
            name: "videos",
            temp: {}
        }
    ]
},
 {
     kind: "pipeline",
     type: "docker",
     name: "promote-" + arch,
     platform: {
         os: "linux",
         arch: arch
     },
     steps: [
     {
             name: "promote",
             image: "debian:buster-slim",
             environment: {
                 AWS_ACCESS_KEY_ID: {
                     from_secret: "AWS_ACCESS_KEY_ID"
                 },
                 AWS_SECRET_ACCESS_KEY: {
                     from_secret: "AWS_SECRET_ACCESS_KEY"
                 }
             },
             commands: [
               "apt update && apt install -y wget",
               "wget https://github.com/syncloud/snapd/releases/download/1/syncloud-release-" + arch + " -O release --progress=dot:giga",
               "chmod +x release",
               "./release promote -n " + name + " -a $(dpkg --print-architecture)"
             ]
       }
      ],
      trigger: {
       event: [
         "promote"
       ]
     }
 }];

build("amd64", true) +
build("arm64", false) +
build("arm", false)
