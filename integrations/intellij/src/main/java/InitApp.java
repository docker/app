import com.intellij.notification.Notification;
import com.intellij.notification.Notifications;
import com.intellij.notification.NotificationType;
import com.intellij.openapi.actionSystem.*;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.ui.Messages;
import com.intellij.ide.util.PropertiesComponent;
import java.util.Vector;
import java.util.Arrays;
import java.io.BufferedReader;
import java.io.File;
import java.io.InputStreamReader;


public class InitApp extends AnAction {
    public InitApp() {
        super("InitApp");
    }
    public void actionPerformed(AnActionEvent event) {
        Project project = event.getProject();
        InitDialog id = new InitDialog();
        id.pack();
        id.setVisible(true);
        if (!id.wasValidated()) {
            return;
        }
        InitDialog.Result r = id.result();
        Vector<String> cmd = new Vector<String>();
        cmd.add("docker-app");
        cmd.add("init");
        cmd.add(r.name);
        if (!r.description.isEmpty()) {
            cmd.add("-d");
            cmd.add(r.description);
        }
        if (!r.maintainers.isEmpty()) {
            String[] mts = r.maintainers.split("\n");
            for (String m: mts) {
                if (!m.isEmpty()) {
                    cmd.add("-m");
                    cmd.add(m);
                }
            }
        }
        Notification no = new Notification("docker-app", "init", cmd.toString(), NotificationType.INFORMATION);
        Notifications.Bus.notify(no);
        try {
            String[] scmd = Arrays.copyOf(cmd.toArray(), cmd.size(), String[].class);
            Process p = Runtime.getRuntime().exec(scmd, null, new File(project.getBasePath()));
            BufferedReader input = new BufferedReader(new InputStreamReader(p.getInputStream()));
            String line;
            while ((line = input.readLine()) != null) {
                Notification n = new Notification("docker-app", "init", line, NotificationType.INFORMATION);
                Notifications.Bus.notify(n);
            }
            BufferedReader error = new BufferedReader(new InputStreamReader(p.getErrorStream()));
            while ((line = error.readLine()) != null) {
                Notification n = new Notification("docker-app", "init", line, NotificationType.ERROR);
                Notifications.Bus.notify(n);
            }
            p.wait();
            String msg = "Application successfuly created.";
            if (p.exitValue()!=0) {
                msg = "Application creation failed, check event log for more informations.";
            }
            Messages.showMessageDialog(project, msg, "Application creation result", Messages.getInformationIcon());

        } catch (Exception e) {
            Messages.showMessageDialog(project, "docker-app invocation failed with " + e.toString(), "Render Failure", Messages.getInformationIcon());
            e.printStackTrace();
        }
    }
}
