using System;
using System.ComponentModel.Design;
using System.Globalization;
using System.IO;
using System.Threading;
using System.Threading.Tasks;
using System.Windows.Forms;
using EnvDTE;
using Microsoft.VisualStudio.Shell;
using Microsoft.VisualStudio.Shell.Interop;
using Task = System.Threading.Tasks.Task;

namespace dockerappvsix
{
    internal sealed class CommandSelectApp
    {
        public const int CommandId = 4129;
        
        public static readonly Guid CommandSet = new Guid("0113e9de-ef33-4d36-9c72-75012c5afd35");
        
        private readonly AsyncPackage _package;
        
        private CommandSelectApp(AsyncPackage package, OleMenuCommandService commandService)
        {
            _package = package ?? throw new ArgumentNullException(nameof(package));
            commandService = commandService ?? throw new ArgumentNullException(nameof(commandService));

            var menuCommandID = new CommandID(CommandSet, CommandId);
            var menuItem = new MenuCommand(this.ExecuteAsync, menuCommandID);
            commandService.AddCommand(menuItem);
        }
        
        public static CommandSelectApp Instance
        {
            get;
            private set;
        }
        
        private Microsoft.VisualStudio.Shell.IAsyncServiceProvider ServiceProvider
        {
            get
            {
                return this._package;
            }
        }
        
        public static async Task InitializeAsync(AsyncPackage package)
        {
            // Verify the current thread is the UI thread - the call to AddCommand in CommandSelectApp's constructor requires
            // the UI thread.
            ThreadHelper.ThrowIfNotOnUIThread();

            OleMenuCommandService commandService = await package.GetServiceAsync((typeof(IMenuCommandService))) as OleMenuCommandService;
            Instance = new CommandSelectApp(package, commandService);
        }
        
        private async void ExecuteAsync(object sender, EventArgs e)
        {
            ThreadHelper.ThrowIfNotOnUIThread();
            DTE dte = await this._package.GetServiceAsync(typeof(DTE)) as DTE;
            Globals g = dte.Solution.Globals;
            string message;
            OpenFileDialog file = new OpenFileDialog();
            file.Title = "Select Docker Application file or metadata file";
            if (file.ShowDialog() == DialogResult.OK)
            {
                string app = file.FileName;
                string f = Path.GetFileName(app);
                if (f == "docker-compose.yml" || f == "parameters.yml" || f == "metadata.yml")
                    app = Path.GetDirectoryName(app);
                message = "Docker Application set to " + app;
                g["dockerapp_applocation"] = app;
            }
            else
            {
                message = "Docker Application unset";
                g["dockerapp_applocation"] = "";
            }
            g.set_VariablePersists("dockerapp_applocation", true);
            string title = "Docker Application selection";

            // Show a message box to prove we were here
            VsShellUtilities.ShowMessageBox(
                this._package,
                message,
                title,
                OLEMSGICON.OLEMSGICON_INFO,
                OLEMSGBUTTON.OLEMSGBUTTON_OK,
                OLEMSGDEFBUTTON.OLEMSGDEFBUTTON_FIRST);
        }
    }
}
