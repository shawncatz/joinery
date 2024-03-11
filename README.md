## Joinery

This application was generated using [Golem](https://github.com/dashotv/golem).

## Getting Started

The application is generated, but doesn't do much by default. To start getting things
running, you'll need to enable some plugins. You can enable a plugin with a command like:

> golem plugin enable [plugin]

### Plugins

Some suggestions to get started.

#### Routes

The `HTTP` server

> golem plugin enable routes

Routes are managed with `groups`. You can add a group with the `golem add group` command. Routes
are then added to the group with the `golem add route` command. The example below adds a group
called `releases` and implements a REST-like interface (Create, Update, etc). Route definitons
are stored in `.golem/routes/`. You can manage them there or you can add addtional routes with
the `golem add route` command.

> golem add group releases --rest

This command generates wrappers (`indexReleasesHandler`, `createReleasesHandler`, etc) to
respond to the following routes. You will need to implement the handler functions
(`ReleaseIndex`, `ReleaseCreate`, etc) in the `app/routes_releases.go` file.

You can always inspect the routes with:

> golem routes

Which will output something like:

```
Releases
         GET /releases/     ReleasesIndex (indexReleasesHandler)
        POST /releases/     ReleasesCreate (createReleasesHandler)
         GET /releases/:id  ReleasesShow (showReleasesHandler)
         PUT /releases/:id  ReleasesUpdate (updateReleasesHandler)
       PATCH /releases/:id  ReleasesSettings (settingsReleasesHandler)
      DELETE /releases/:id  ReleasesDelete (deleteReleasesHandler)
```


#### Models

The `Database` access layer.

> golem plugin enable models

Models are managed with the `golem add model` command. The example below adds a model
called `release` with a `name` field. Model definitions are stored in `.golem/models/`.
You can manage the models in the yaml files in this directory.

> golem add model release name:string

All the generated models are in the `app/app_models.go` file. A file is generated for each
model for customizing the functionality of the generated model. In the example above, this
creates a file called `app/models_release.go`. You can add custom methods to the model in
this file.

Database access is enabled by the [grimoire](https://github.com/dashotv/grimoire) package.
It provides a simple interface to creating, saving, and querying data. You can find more
information at the link above. `Grimoire` is a layer built on
[kamva/mgm](https://github.com/kamva/mgm).

#### Events

Event processing and handling. Manage channels and event handlers that interact with other
services using event-based communications.

> golem plugin enable events

Events are managed with the `golem add event` command. The example below adds an event
called `release` with a `name` field. Event definitions are stored in `.golem/events/`.
You can manage the events in the yaml files in this directory.

> golem add event release name:string

All of the generated channels and events are in the `app/app_events.go` file. A file is
generated for each event for implementing the handling functionality. In the example
above, this creates a file called `app/events_release.go`.

There are ultimately three types of events:
* `Sender` - allows you to send a message to another service
* `Receiver` - allows you to receive a message from another service
* `Receiver` with `Proxy` enabled - this is a combination of the two. It
  allows you to receive a message, process it and pass it on to another.

Both `Receeiver` types of events have generated `handler` function called `on[Name]`. The
`Receiver` handler just allows you to process the message. The `Receiver` with `Proxy` enabled
handler allows you to process the message and then return a different type.

The `Events` plugin automatically wires all of the encoding/decoding (JSON) and sending/receiving
functionality using a lightweight wrapper around the [NATS](https://nats.io/) messaging system.
This wrapper is [Mercury](https://github.com/dashotv/mercury)

#### Workers

Background jobs

> golem plugin enable workers

Workers are managed with the `golem add worker` command. The example below adds a worker
called `ReleaseProcess`. Worker definitions are stored in `.golem/workers/`. You can manage
the workers in the yaml files in this directory.

> golem add worker ReleaseProcess

All of the generated worker management code is in the `app/app_workers.go` file. A file
is generated for each worker for implementing the work functionality. In the example above,
this creates a file called `app/workers_release_process.go`.

The `Workers` plugin automatically wires all of the worker management functionality using
the [Minion](https://github.com/dashotv/minion) package.

#### Cache

Cache management

> golem plugin enable cache

Cache is a simple wrapper around the `redis` implementation of
[gokv](https://github.com/philippgille/gokv).

The wrapper provides a `Fetch` method that works similarly to the `Rails.cache.fetch`
method. It allows you to fetch a value from the cache and if it doesn't exist, it will
call a function to generate the value and store it in the cache.

