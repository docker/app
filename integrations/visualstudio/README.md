# Docker Application extension for Visual Studio

This directory contains the sources of a Docker Application extension for Visual Studio.

The extension creates a few commands in the `Tools` menu.

![The extension menu](vsextensionscreenshot.png)
# Restoring dependencies

Run `nuget.exe restore`.

# Building the extension

Open the `dockerappvsix.sln` solution in Visual Studio, change the target to `Release` and hit build. This will produce the extension under `dockerappvsix/bin/Release/dockerappvsix.vsix`.

# Installing the extension

Double-click on the `dockerappsvix.vsix` file in the explorer. This will prompt you to install the extension.

# Using the extension

The extension exposes the following commands in the `Tools` menu:

The extension requires that the `docker-app` tool is in your `$PATH`.

## New application

This command displays a dialog that can be used to initialize a new Docker Application.

It gives you the option to choose the name, description and maintainers of the application, as well as whether to use single-file mode or not.

## Select application

By default all operations will look for a single Docker Application at the root of the solution directory. If your application is located elsewhere, or if you have multiple applications, you can use the `select application` menu to select which application will be used.

## Render

`Render` simply renders the application in a popup window.

## Parameters

`Parameters` pops-up a dialog that can be used to configure deployment parameters, such as which orchestrator to use, the stack name and namespace, and parameters overrides.

## Deploy

`Deploy` deploys your application to a cluster. Progress or eventual errors are displayed in the event log.
