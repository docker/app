using System;
using System.ComponentModel.Design;
using System.Globalization;
using System.IO;
using System.Threading;
using System.Threading.Tasks;
using System.Windows.Controls;
using EnvDTE;
using Microsoft.VisualStudio.PlatformUI;
using Microsoft.VisualStudio.Shell;
using Microsoft.VisualStudio.Shell.Interop;
using Task = System.Threading.Tasks.Task;

namespace dockerappvsix
{

    class NewDialog : DialogWindow
    {
        public class Settings
        {
            public string name;
            public string description;
            public string maintainers;
            public bool singleFile;
        };
        Grid grid;
        TextBox tName;
        TextBox tDescription;
        TextBox tMaintainers;
        CheckBox cSingleFile;
        Button bOk;
        Button bCancel;
        bool validated;
        public Settings Get()
        {
            return new Settings
            {
                name = tName.Text,
                description = tDescription.Text,
                maintainers = tMaintainers.Text,
                singleFile = cSingleFile.IsChecked ?? false
            };
        }
        private void AddGrid(Grid g, System.Windows.UIElement el, int r, int c)
        {
            Grid.SetRow(el, r);
            Grid.SetColumn(el, c);
            g.Children.Add(el);
        }
        public NewDialog()
        {
            this.HasMaximizeButton = true;
            this.HasMinimizeButton = true;
            validated = false;
            grid = new Grid();
            grid.RowDefinitions.Add(new RowDefinition());
            grid.RowDefinitions.Add(new RowDefinition());
            grid.RowDefinitions.Add(new RowDefinition());
            grid.RowDefinitions.Add(new RowDefinition());
            grid.RowDefinitions.Add(new RowDefinition());
            grid.ColumnDefinitions.Add(new ColumnDefinition());
            grid.ColumnDefinitions.Add(new ColumnDefinition());
       
            tName = new TextBox();
            AddGrid(grid, new Label { Content = "Name" }, 0, 0);
            AddGrid(grid, tName, 0, 1);
            tDescription = new TextBox();
            AddGrid(grid, new Label { Content = "Description" }, 1, 0);
            AddGrid(grid, tDescription, 1, 1);
            tMaintainers = new TextBox { MinLines = 3, AcceptsReturn = true };
            AddGrid(grid, new Label { Content = "Maintainers" }, 2, 0);
            AddGrid(grid, tMaintainers, 2, 1);
            cSingleFile = new CheckBox { Content = "single file" };
            AddGrid(grid, cSingleFile, 3, 0);
            bOk = new Button { Content = "OK" };
            bCancel = new Button { Content = "Cancel" };
            AddGrid(grid, bOk, 4, 0);
            AddGrid(grid, bCancel, 4, 1);
            bOk.Click += BOk_Click;
            bCancel.Click += BCancel_Click;
            this.Content = grid;
        }

        private void BCancel_Click(object sender, System.Windows.RoutedEventArgs e)
        {
            validated = false;
            Close();
        }
        private void BOk_Click(object sender, System.Windows.RoutedEventArgs e)
        {
            validated = true;
            Close();
        }
       
        public bool Validated()
        {
            return validated;
        }
    }
    /// <summary>
    /// Command handler
    /// </summary>
    internal sealed class CommandNew
    {
        /// <summary>
        /// Command ID.
        /// </summary>
        public const int CommandId = 4132;

        /// <summary>
        /// Command menu group (command set GUID).
        /// </summary>
        public static readonly Guid CommandSet = new Guid("0113e9de-ef33-4d36-9c72-75012c5afd35");

        /// <summary>
        /// VS Package that provides this command, not null.
        /// </summary>
        private readonly AsyncPackage package;

        /// <summary>
        /// Initializes a new instance of the <see cref="CommandNew"/> class.
        /// Adds our command handlers for menu (commands must exist in the command table file)
        /// </summary>
        /// <param name="package">Owner package, not null.</param>
        /// <param name="commandService">Command service to add command to, not null.</param>
        private CommandNew(AsyncPackage package, OleMenuCommandService commandService)
        {
            this.package = package ?? throw new ArgumentNullException(nameof(package));
            commandService = commandService ?? throw new ArgumentNullException(nameof(commandService));

            var menuCommandID = new CommandID(CommandSet, CommandId);
            var menuItem = new MenuCommand(this.ExecuteAsync, menuCommandID);
            commandService.AddCommand(menuItem);
        }

        /// <summary>
        /// Gets the instance of the command.
        /// </summary>
        public static CommandNew Instance
        {
            get;
            private set;
        }

        /// <summary>
        /// Gets the service provider from the owner package.
        /// </summary>
        private Microsoft.VisualStudio.Shell.IAsyncServiceProvider ServiceProvider
        {
            get
            {
                return this.package;
            }
        }

        /// <summary>
        /// Initializes the singleton instance of the command.
        /// </summary>
        /// <param name="package">Owner package, not null.</param>
        public static async Task InitializeAsync(AsyncPackage package)
        {
            // Verify the current thread is the UI thread - the call to AddCommand in CommandNew's constructor requires
            // the UI thread.
            ThreadHelper.ThrowIfNotOnUIThread();

            OleMenuCommandService commandService = await package.GetServiceAsync((typeof(IMenuCommandService))) as OleMenuCommandService;
            Instance = new CommandNew(package, commandService);
        }

        /// <summary>
        /// This function is the callback used to execute the command when the menu item is clicked.
        /// See the constructor to see how the menu item is associated with this function using
        /// OleMenuCommandService service and MenuCommand class.
        /// </summary>
        /// <param name="sender">Event sender.</param>
        /// <param name="e">Event args.</param>
        private async void ExecuteAsync(object sender, EventArgs e)
        {
            ThreadHelper.ThrowIfNotOnUIThread();
            NewDialog ns = new NewDialog();
            ns.ShowModal();
            if (!ns.Validated())
                return;
            NewDialog.Settings s = ns.Get();
            string args = "init " + s.name;
            if (s.description != "")
                args += " --description \"" + s.description + "\"";
            if (s.maintainers != "")
            {
                foreach (string m in s.maintainers.Split('\n'))
                {
                    args += " --maintainer \"" + m + "\"";
                }
            }
            if (s.singleFile)
                args += " -s";
            System.Diagnostics.Process proc = new System.Diagnostics.Process();
            proc.StartInfo.FileName = "docker-app";
            proc.StartInfo.UseShellExecute = false;
            proc.StartInfo.RedirectStandardError = true;
            proc.StartInfo.RedirectStandardOutput = true;
            proc.StartInfo.Arguments = args;
            DTE dte = await this.package.GetServiceAsync(typeof(DTE)) as DTE;        
            if (dte.Solution.FileName != "")
            {
                string wd = Path.GetDirectoryName(dte.Solution.FileName);
                proc.StartInfo.WorkingDirectory = wd;
            }
            string message;
            proc.Start();
            proc.WaitForExit();
            string serr = proc.StandardError.ReadToEnd();
            string sout = proc.StandardOutput.ReadToEnd();
            message = serr + sout;
            if (proc.ExitCode != 0)
                message = "Error creating application:" + System.Environment.NewLine + message;
            else
                message = "Application created!" + System.Environment.NewLine + message;
            string title = "Create Application";
            // Show a message box to prove we were here
            VsShellUtilities.ShowMessageBox(
                this.package,
                message,
                title,
                OLEMSGICON.OLEMSGICON_INFO,
                OLEMSGBUTTON.OLEMSGBUTTON_OK,
                OLEMSGDEFBUTTON.OLEMSGDEFBUTTON_FIRST);
        }
    }
}
