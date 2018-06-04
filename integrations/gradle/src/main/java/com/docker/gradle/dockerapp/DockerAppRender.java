package com.docker.gradle.dockerapp;

import org.gradle.api.DefaultTask;
import org.gradle.api.tasks.Input;
import org.gradle.api.tasks.OutputFile;
import org.gradle.api.tasks.TaskAction;

import java.io.BufferedReader;
import java.io.InputStreamReader;

public class DockerAppRender extends DefaultTask {
    private String appPath;
    private String target;
    @OutputFile
    public String GetTarget() {
        return target;
    }
    public void setTarget(String t) {
        target = t;
    }
    @Input
    public String GetAppPath() {
        return appPath;
    }

    public void setAppPath(String ap) {
        appPath = ap;
    }
    @TaskAction
    public void process() {
        try {
            Process p = Runtime.getRuntime().exec("docker-app " + "render " + appPath + " -o " + target);
            BufferedReader input =
                    new BufferedReader
                            (new InputStreamReader(p.getInputStream()));
            String line;
            while ((line = input.readLine()) != null) {
                System.out.println(line);
            }
            input.close();
        } catch (Exception e) {
             e.printStackTrace();
        }
    }
}
