# Traefik Plugin: Response Rewrite Body

[![License](https://img.shields.io/badge/license-Apache2-blue.svg)](https://github.com/quortex/traefik-responsebodyrewrite/blob/main/LICENSE)

This repo contains a Traefik plugin that allows you to rewrite response body based on a parameters.

## Configuration

It is possible to install the [plugin locally](https://traefik.io/blog/using-private-plugins-in-traefik-proxy-2-5/) or to install it through [Traefik Pilot](https://pilot.traefik.io/plugins).

### Configuration as local plugin

Depending on your setup, the installation steps might differ from the one described here. This example assumes that your Traefik instance runs in a Docker container and uses the [official image](https://hub.docker.com/_/traefik/).

Download the latest release of the plugin and save it to a location the Traefik container can reach. Below is an example of a possible setup. Notice how the plugin source is mapped into the container (`/plugin/traefik-responsebodyrewrite:/plugins-local/src/github.com/quortex/traefik-responsebodyrewrite/`) via a volume bind mount:

#### `docker-compose.yml`

````yml
version: "3.7"

services:
  traefik:
    image: traefik

    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /docker/config/traefik/traefik.yml:/etc/traefik/traefik.yml
      - /docker/config/traefik/dynamic-configuration.yml:/etc/traefik/dynamic-configuration.yml
      - /docker/config/traefik/plugin/traefik-responsebodyrewrite:/plugins-local/src/github.com/quortex/traefik-responsebodyrewrite/

    ports:
      - "8080:80"

  hello:
    image: ealen/echo-server
    labels:
      - traefik.enable=true
      - traefik.http.routers.hello.entrypoints=http
      - traefik.http.routers.hello.rule=Host(`localhost`)
      - traefik.http.services.hello.loadbalancer.server.port=80
      - traefik.http.routers.hello.middlewares=my-traefik-responsebodyrewrite@file
      
````

To complete the setup, the Traefik configuration must be extended with the plugins. For this you must create the `traefik.yml` and the dynamic-configuration.yml` files if not present already.

````yml
log:
  level: INFO

experimental:
  localPlugins:
    traefik-responsebodyrewrite:
      moduleName: github.com/quortex/traefik-responsebodyrewrite
````

#### `dynamic-configuration.yml`

````yml
http:
  middlewares:
    my-traefik-responsebodyrewrite:
      plugin:
        traefik-responsebodyrewrite:
          responses:
            - status: 200
              rewrites:
                - regex: foo
                  replacement: "Bar"
            - status: 400-499
              rewrites:
                - regex: .*
                  replacement: "Error Replacement"

 
````
### Traefik Plugin registry

This procedure will install the plugins via the [Traefik Plugin registry](https://plugins.traefik.io/install).

Add the following code to your `traefik-config.yml`

```yml
experimental:
  plugins:
    traefik-responsebodyrewrite:
      moduleName: "github.com/quortex/traefik-responsebodyrewrite"
      version: "v0.0.0"
# other stuff you might have in your traefik-config
entryPoints:
  http:
    address: ":80"
  https:
    address: ":443"

providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
  file:
    filename: "/etc/traefik/dynamic-configuration.yml"
```

In your dynamic configuration add the following code:

```yml
http:
  middlewares:
    my-traefik-responsebodyrewrite:
      plugin:
        traefik-responsebodyrewrite:
          responses:
            - status: 200
              rewrites:
                - regex: foo
                  replacement: "Bar"
            - status: 400-499
              rewrites:
                - regex: .*
                  replacement: "Error Replacement"

```

## Develop
A docker compose configuration is already sets to run a traefik and and echo server with local plugin deployed 
```bash
docker compose -f docker/dev/docker-compose.yml up
```

### Testing body rewrite
You can run a curl to check the response body
```bash
curl  -v "http://localhost:8080?echo_body=foobrady&echo_code=200"
curl  -v "http://localhost:8080?echo_body=foobrady&echo_code=404"
curl  -v "http://localhost:8080?echo_body=foobrady&echo_code=500"
```
- 200 should return `Barbrady`
- 404 should return `Error Replacement`
- 500 should not be modified and return `Barbrady`

