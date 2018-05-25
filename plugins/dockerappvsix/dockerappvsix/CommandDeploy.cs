using System;
using System.ComponentModel.Design;
using System.Globalization;
using System.IO;
using System.Threading;
using System.Threading.Tasks;
using EnvDTE;
using Microsoft.VisualStudio;
using Microsoft.VisualStudio.Shell;
using Microsoft.VisualStudio.Shell.Interop;
using Task = System.Threading.Tasks.Task;

namespace dockerappvsix
{
    /// <summary>
    /// Command handler
    /// </summary>
    internal sealed class CommandDeploy
    {
        /// <summary>
        /// Command ID.
        /// </summary>
        public const int CommandId = 4131;

        /// <summary>
        /// Command menu group (command set GUID).
        /// </summary>
        public static readonly Guid CommandSet = new Guid("0113e9de-ef33-4d36-9c72-75012c5afd35");

        /// <summary>
        /// VS Package that provides this command, not null.
        /// </summary>
        private readonly AsyncPackage package;

        /// <summary>
        /// Initializes a new instance of the <see cref="CommandDeploy"/> class.
        /// Adds our command handlers for menu (commands must exist in the command table file)
        /// </summary>
        /// <param name="package">Owner package, not null.</param>
        /// <param name="commandService">Command service to add command to, not null.</param>
        private CommandDeploy(AsyncPackage package, OleMenuCommandService commandService)
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
        public static CommandDeploy Instance
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
            // Verify the current thread is the UI thread - the call to AddCommand in CommandDeploy's constructor requires
            // the UI thread.
            ThreadHelper.ThrowIfNotOnUIThread();

            OleMenuCommandService commandService = await package.GetServiceAsync((typeof(IMenuCommandService))) as OleMenuCommandService;
            Instance = new CommandDeploy(package, commandService);
        }

        private string AddArg(Globals g, string key, string cmd)
        {
            if (g.get_VariableExists(key) && g[key] as string != "")
                return " " + cmd + " " + g[key];
            return "";
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
            DTE dte = await this.package.GetServiceAsync(typeof(DTE)) as DTE;
            Globals g = dte.Solution.Globals;
            string args = "deploy";
            args += AddArg(g, "dockerapp_applocation", "");
            args += AddArg(g, "dockerapp_orchestrator", "--orchestrator");
            args += AddArg(g, "dockerapp_stackname", "--name");
            args += AddArg(g, "dockerapp_namespace", "--namespace");
            args += AddArg(g, "dockerapp_kubeconfig", "--kubeconfig");
            if (g.get_VariableExists("dockerapp_settings"))
            {
                foreach (string s in (g["dockerapp_settings"] as string).Split('\n'))
                    args += " -s " + s;
            }
            System.Diagnostics.Process proc = new System.Diagnostics.Process();
            proc.StartInfo.FileName = "docker-app";
            proc.StartInfo.UseShellExecute = false;
            proc.StartInfo.RedirectStandardError = true;
            proc.StartInfo.RedirectStandardOutput = true;
            proc.StartInfo.Arguments = args;

            if (dte.Solution.FileName != "")
            {
                string wd = Path.GetDirectoryName(dte.Solution.FileName);
                proc.StartInfo.WorkingDirectory = wd;
            }
            proc.Start();
            IVsOutputWindow outWindow = Package.GetGlobalService(typeof(SVsOutputWindow)) as IVsOutputWindow;

            Guid generalPaneGuid = VSConstants.GUID_OutWindowDebugPane; //  GUID_OutWindowGeneralPane fails on vs2017
            IVsOutputWindowPane generalPane;
            outWindow.GetPane(ref generalPaneGuid, out generalPane);

            generalPane.OutputString("Deploy command: docker-app " + args + System.Environment.NewLine);
            generalPane.Activate(); // Brings this pane into view
            while (!proc.StandardOutput.EndOfStream)
                generalPane.OutputString(proc.StandardOutput.ReadLine() + System.Environment.NewLine);
            while (!proc.StandardError.EndOfStream)
                generalPane.OutputString(proc.StandardError.ReadLine() + System.Environment.NewLine);
        }
    }
}
