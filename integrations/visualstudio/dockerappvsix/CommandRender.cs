using System;
using System.ComponentModel.Design;
using System.Diagnostics;
using System.Globalization;
using System.IO;
using System.Reflection;
using System.Threading;
using System.Threading.Tasks;
using EnvDTE;
using Microsoft.VisualStudio.Parameters;
using Microsoft.VisualStudio.Shell;
using Microsoft.VisualStudio.Shell.Interop;
using Task = System.Threading.Tasks.Task;

namespace dockerappvsix
{
    internal sealed class CommandRender
    {
        public const int CommandId = 0x0100;
        
        public static readonly Guid CommandSet = new Guid("0113e9de-ef33-4d36-9c72-75012c5afd35");
        
        private readonly AsyncPackage _package;
        
        private CommandRender(AsyncPackage package, OleMenuCommandService commandService)
        {
            _package = package ?? throw new ArgumentNullException(nameof(package));
            commandService = commandService ?? throw new ArgumentNullException(nameof(commandService));

            var menuCommandID = new CommandID(CommandSet, CommandId);
            var menuItem = new MenuCommand(this.ExecuteAsync, menuCommandID);
            commandService.AddCommand(menuItem);
        }
        
        public static CommandRender Instance
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
            // Verify the current thread is the UI thread - the call to AddCommand in Command1's constructor requires
            // the UI thread.
            ThreadHelper.ThrowIfNotOnUIThread();

            OleMenuCommandService commandService = await package.GetServiceAsync((typeof(IMenuCommandService))) as OleMenuCommandService;
            Instance = new CommandRender(package, commandService);
        }
        
        private async void ExecuteAsync(object sender, EventArgs e)
        {
            ThreadHelper.ThrowIfNotOnUIThread();
            System.Diagnostics.Process proc = new System.Diagnostics.Process();
            proc.StartInfo.FileName = "docker-app";
            proc.StartInfo.UseShellExecute = false;
            proc.StartInfo.RedirectStandardError = true;
            proc.StartInfo.RedirectStandardOutput = true;
            proc.StartInfo.Arguments = "render";
            DTE dte = await this._package.GetServiceAsync(typeof(DTE)) as DTE;
            Globals g = dte.Solution.Globals;
            var appLocation = g.GetOrNull<string>("dockerapp_applocation");
            if (!string.IsNullOrEmpty(appLocation))
            {
                proc.StartInfo.Arguments += " " + appLocation;
            }
            if (dte.Solution.FileName != "")
            {
                string wd = Path.GetDirectoryName(dte.Solution.FileName);
                proc.StartInfo.WorkingDirectory = wd;
            }
            string message;
            try
            {
                proc.Start();
                proc.WaitForExit();
                string serr = proc.StandardError.ReadToEnd();
                string sout = proc.StandardOutput.ReadToEnd();
                message = serr + sout;
            } catch (Exception ex)
            {
                message = "Cannot run docker-app: " + ex.ToString();
            }
            
            string title = "Docker Application Render";

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
