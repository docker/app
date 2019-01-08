import com.intellij.openapi.actionSystem.*;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.ui.Messages;
import com.intellij.ide.util.PropertiesComponent;

import java.io.File;
import java.util.Scanner;


public class RenderApp extends AnAction {
    public RenderApp() {
        super("RenderApp");
    }

    public void actionPerformed(AnActionEvent event) {
        Project project = event.getProject();
        PropertiesComponent pc = PropertiesComponent.getInstance(project);
        String appPath = pc.getValue("docker_app_path");
        try {
            String rawParameters = pc.getValue("docker_app_overrides");
            String parameters = "";
            if (!rawParameters.isEmpty()) {
                String[] split = rawParameters.split("\n");
                for (String l: split) {
                    parameters += " -s " + l;
                }
            }
            Process p = Runtime.getRuntime().exec("docker-app render " + appPath + parameters, null, new File(project.getBasePath()));
            Scanner se = new Scanner(p.getErrorStream()).useDelimiter("\\A");
            String stderr = se.hasNext() ? se.next() : "";
            Scanner so = new Scanner(p.getInputStream()).useDelimiter("\\A");
            String stdout = so.hasNext() ? so.next() : "";
            Messages.showMessageDialog(project, stderr + stdout, "Rendered Application", Messages.getInformationIcon());
         } catch (Exception e) {
            Messages.showMessageDialog(project, "docker-app invocation failed with " + e.toString(), "Render Failure", Messages.getInformationIcon());
            e.printStackTrace();
        }

    }
}
