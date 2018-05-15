# DockerApp gradle plugin

This directory contains a gradle plugin that exposes tasks that perform various docker-app related operations.

# Building the plugin

Running `gradle build` in the plugin directory will create the plugin jar file in the `build/lib` directory.

# Using the plugin

Put the following code into your build.gradle file:

    buildscript{
        dependencies{
            classpath files('path/to/dockerapp-plugin-1.0-SNAPSHOT.jar')
        }
    }

    apply plugin: com.docker.gradle.dockerapp.DockerAppPlugin

    import com.docker.gradle.dockerapp.*

You can then use the tasks exposed by this plugin:

## Render

This task performs a `docker-app render`.

    task renderMyApp(type: DockerAppRender) {
      appPath = 'path/to/dockerapp'
      target = 'rendered-output.yml'
    }
