# DockerApp Gradle plugin

This directory contains an experimental Gradle plugin that exposes tasks that make `docker-app` available via the Gradle build tool. This in turn makes it easy to integrate with your IDE of choice.

## Building the plugin

Running `./gradlew build` in the plugin directory will create the plugin jar file in the `build/lib` directory.

## Using the plugin

Put the following code into your build.gradle file:

```gradle
buildscript{
    dependencies{
        classpath files('path/to/dockerapp-plugin-1.0-SNAPSHOT.jar')
    }
}

apply plugin: com.docker.gradle.dockerapp.DockerAppPlugin

import com.docker.gradle.dockerapp.*
```

You can then use the tasks exposed by this plugin:

### Render

This task performs a `docker-app render`.

```gradle
task renderMyApp(type: DockerAppRender) {
  appPath = 'path/to/dockerapp'
  target = 'rendered-output.yml'
}
```
