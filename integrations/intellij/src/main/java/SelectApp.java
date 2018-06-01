import com.intellij.openapi.actionSystem.*;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.ui.Messages;
import com.intellij.openapi.vfs.*;
import com.intellij.openapi.fileChooser.*;
import com.intellij.ide.util.PropertiesComponent;

public class SelectApp extends AnAction {
    public SelectApp() {
        super("SelectApp");
    }

    public void actionPerformed(AnActionEvent event) {
        Project project = event.getProject();
        VirtualFile  vf = FileChooser.chooseFile(new FileChooserDescriptor(true, true, false, false, false, false), project, null);
        String path = "";
        String msg = "";
        if (vf != null) {
            path = vf.getCanonicalPath();
            msg = "Application path set to " + path;

        } else {
            msg = "Application path unset";
        }
        PropertiesComponent.getInstance(project).setValue("docker_app_path", path);
        Messages.showMessageDialog(project,  msg, "Confirmation", Messages.getInformationIcon());
    }
}
