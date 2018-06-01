using Microsoft.VisualStudio.PlatformUI;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Documents;
using System.Windows.Input;
using System.Windows.Media;
using System.Windows.Media.Imaging;
using System.Windows.Navigation;
using System.Windows.Shapes;

namespace dockerappvsix
{
    /// <summary>
    /// Interaction logic for SettingsDialog.xaml
    /// </summary>
    public partial class SettingsDialog : DialogWindow
    {
        public SettingsDialog()
        {
            InitializeComponent();
        }

        public AppPackageSettings Settings
        {
            get
            {
                return DataContext as AppPackageSettings;
            }
        }

        private void OkClick(object sender, RoutedEventArgs e)
        {
            DialogResult = true;
            Close();
        }

        private void CancelClick(object sender, RoutedEventArgs e)
        {
            DialogResult = false;
            Close();
        }

        private void OnBrowse(object sender, RoutedEventArgs e)
        {
            var baseFile = Settings.KubeConfig;
            var ofd = new Microsoft.Win32.OpenFileDialog();
            ofd.FileName = baseFile;
            if (ofd.ShowDialog() ?? false) {
                Settings.KubeConfig = ofd.FileName;
            }
        }
    }
}
