import com.intellij.openapi.actionSystem.*;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.ui.*;
import com.intellij.openapi.ui.popup.*;

public class Parameters extends AnAction {
    public Parameters() { super("Parameters"); }

    public void actionPerformed(AnActionEvent event) {
        ParametersDialog sf = new ParametersDialog();
        sf.pack();
        sf.load(event.getProject());
        sf.setVisible(true);
        sf.save(event.getProject());
    }
}
