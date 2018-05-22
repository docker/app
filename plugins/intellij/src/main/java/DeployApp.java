import com.intellij.notification.Notification;
import com.intellij.notification.Notifications;
import com.intellij.notification.NotificationType;
import com.intellij.openapi.actionSystem.*;
import com.intellij.openapi.diagnostic.FrequentEventDetector;
import com.intellij.openapi.diagnostic.Logger;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.ui.Messages;
import com.intellij.ide.util.PropertiesComponent;
import org.apache.log4j.Level;

import java.io.BufferedReader;
import java.io.File;
import java.io.InputStreamReader;
import java.util.Scanner;


public class DeployApp extends AnAction {
    public DeployApp() {
        super("DeployApp");
    }

    public void actionPerformed(AnActionEvent event) {
        Project project = event.getProject();
        PropertiesComponent pc = PropertiesComponent.getInstance(project);
        String appPath = pc.getValue("docker_app_path");
        if (appPath == null || appPath.equals("")) {
            appPath = ""; // project.getBasePath();
        }
        try {
            String orchestrator = "swarm";
            if (pc.getValue("docker_app_orchestrator").equals("kubernetes"))
                orchestrator = "kubernetes";
            String rawSettings = pc.getValue("docker_app_overrides");
            String settings = "";
            if (!rawSettings.isEmpty()) {
                String[] split = rawSettings.split("\n");
                for (String l: split) {
                    settings += " -s " + l;
                }
            }
            String kubeconfig = pc.getValue("docker_app_kubeconfig");
            if (!kubeconfig.isEmpty()) {
                kubeconfig = " --kubeconfig " + kubeconfig;
            }
            String namespace = pc.getValue("docker_app_namespace");
            if (!namespace.isEmpty()) {
                namespace = " --namespace " + namespace;
            }
            String name = pc.getValue("docker_app_name");
            if (!name.isEmpty()) {
                name = " --name " + name;
            }
            String cmd = "docker-app deploy " + appPath
                    + " --orchestrator="+orchestrator
                    + kubeconfig
                    + namespace
                    + name
                    + settings;
            Process p = Runtime.getRuntime().exec(cmd,null, new File(project.getBasePath()));
            /*Logger l = Logger.getInstance("docker-app");
            l.setLevel(Level.INFO);
            l.info("Running deploy command: " + cmd);*/
            BufferedReader input = new BufferedReader(new InputStreamReader(p.getInputStream()));
            String line;
            while ((line = input.readLine()) != null) {
                Notification n = new Notification("docker-app", "deploy", line, NotificationType.INFORMATION);
                Notifications.Bus.notify(n);
                //l.info(line);
            }
            BufferedReader error = new BufferedReader(new InputStreamReader(p.getErrorStream()));
            while ((line = error.readLine()) != null) {
                Notification n = new Notification("docker-app", "deploy", line, NotificationType.ERROR);
                Notifications.Bus.notify(n);
                //l.warn(line);
            }
        } catch (Exception e) {
            Messages.showMessageDialog(project, "docker-app invocation failed with " + e.toString(), "Render Failure", Messages.getInformationIcon());
            e.printStackTrace();
        }

    }
}
