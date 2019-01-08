import javax.swing.*;
import java.awt.event.*;
import com.intellij.openapi.project.Project;
import com.intellij.ide.util.PropertiesComponent;

public class ParametersDialog extends JDialog {
    private JPanel contentPane;
    private JButton buttonOK;
    private JButton buttonCancel;
    private JTextField tKubeconfig;
    private JTextArea tOverrides;
    private JRadioButton oSwarm;
    private JRadioButton oKubernetes;
    private JTextField tNamespace;
    private JTextField tStackName;
    private boolean validated;

    public ParametersDialog() {
        setContentPane(contentPane);
        setModal(true);
        getRootPane().setDefaultButton(buttonOK);

        buttonOK.addActionListener(new ActionListener() {
            public void actionPerformed(ActionEvent e) {
                onOK();
            }
        });

        buttonCancel.addActionListener(new ActionListener() {
            public void actionPerformed(ActionEvent e) {
                onCancel();
            }
        });

        // call onCancel() when cross is clicked
        setDefaultCloseOperation(DO_NOTHING_ON_CLOSE);
        addWindowListener(new WindowAdapter() {
            public void windowClosing(WindowEvent e) {
                onCancel();
            }
        });

        // call onCancel() on ESCAPE
        contentPane.registerKeyboardAction(new ActionListener() {
            public void actionPerformed(ActionEvent e) {
                onCancel();
            }
        }, KeyStroke.getKeyStroke(KeyEvent.VK_ESCAPE, 0), JComponent.WHEN_ANCESTOR_OF_FOCUSED_COMPONENT);
    }

    public void load(Project project) {
        PropertiesComponent pc = PropertiesComponent.getInstance(project);
        String orchestrator = pc.getValue("docker_app_orchestrator");
        if (orchestrator != null && orchestrator.equals("kubernetes")) {
            oKubernetes.setSelected(true);
        } else {
            oSwarm.setSelected(true);
        }
        tKubeconfig.setText(pc.getValue("docker_app_kubeconfig"));
        tOverrides.setText(pc.getValue("docker_app_overrides"));
        tNamespace.setText(pc.getValue("docker_app_namespace"));
        tStackName.setText(pc.getValue("docker_app_name"));
    }
    public void save(Project project) {
        if (!validated)
            return;
        PropertiesComponent pc = PropertiesComponent.getInstance(project);
        pc.setValue("docker_app_orchestrator", oKubernetes.isSelected()? "kubernetes" : "swarm");
        pc.setValue("docker_app_kubeconfig", tKubeconfig.getText());
        pc.setValue("docker_app_overrides", tOverrides.getText());
        pc.setValue("docker_app_namespace", tNamespace.getText());
        pc.setValue("docker_app_name", tStackName.getText());
    }
    private void onOK() {
        // add your code here
        validated = true;
        dispose();
    }

    private void onCancel() {
        // add your code here if necessary
        validated = false;
        dispose();
    }
}
