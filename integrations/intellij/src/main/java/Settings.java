import com.intellij.openapi.actionSystem.*;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.ui.*;
import com.intellij.openapi.ui.popup.*;

public class Settings extends AnAction {
    public Settings() { super("Settings"); }

    public void actionPerformed(AnActionEvent event) {
        SettingsDialog sf = new SettingsDialog();
        sf.pack();
        sf.load(event.getProject());
        sf.setVisible(true);
        sf.save(event.getProject());
    }
}
