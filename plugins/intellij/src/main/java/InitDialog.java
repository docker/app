import javax.swing.*;
import java.awt.event.*;

public class InitDialog extends JDialog {
    private JPanel contentPane;
    private JButton buttonOK;
    private JButton buttonCancel;
    private JTextField tName;
    private JTextField tDescription;
    private JTextArea tMaintainers;
    private JCheckBox cSingleFile;
    private boolean validated;

    public class Result {
        public String name;
        public String description;
        public String maintainers;
        public boolean singleFile;
    }

    public boolean wasValidated() { return validated;}

    public Result result() {
        Result r = new Result();
        r.name = tName.getText();
        r.description = tDescription.getText();
        r.maintainers = tMaintainers.getText();
        r.singleFile = cSingleFile.isSelected();
        return r;
    }

    public InitDialog() {
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
