using System;
using System.ComponentModel.Design;
using System.Globalization;
using System.IO;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using EnvDTE;
using Microsoft.VisualStudio;
using Microsoft.VisualStudio.Shell;
using Microsoft.VisualStudio.Shell.Interop;
using Task = System.Threading.Tasks.Task;

namespace dockerappvsix
{
    internal sealed class CommandDeploy
    {
        public const int CommandId = 4131;
        
        public static readonly Guid CommandSet = new Guid("0113e9de-ef33-4d36-9c72-75012c5afd35");
        
        private readonly AsyncPackage _package;

        private CommandDeploy(AsyncPackage package, OleMenuCommandService commandService)
        {
            _package = package ?? throw new ArgumentNullException(nameof(package));
            commandService = commandService ?? throw new ArgumentNullException(nameof(commandService));

            var menuCommandID = new CommandID(CommandSet, CommandId);
            var menuItem = new MenuCommand(this.ExecuteAsync, menuCommandID);
            commandService.AddCommand(menuItem);
        }
        
        public static CommandDeploy Instance
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
            ThreadHelper.ThrowIfNotOnUIThread();

            OleMenuCommandService commandService = await package.GetServiceAsync((typeof(IMenuCommandService))) as OleMenuCommandService;
            Instance = new CommandDeploy(package, commandService);
        }

        private void AddArgIfExists(Globals g, string key, string flag, StringBuilder cmd)
        {
            var value = g.GetOrNull<string>(key);
            if (string.IsNullOrEmpty(value)) {
                return;
            }
            
            if (!string.IsNullOrEmpty(flag)) {
                cmd.Append($" {flag}");
            }
            cmd.Append($" {value}");
        }
        private async void ExecuteAsync(object sender, EventArgs e)
        {
            ThreadHelper.ThrowIfNotOnUIThread();
            DTE dte = await this._package.GetServiceAsync(typeof(DTE)) as DTE;
            Globals g = dte.Solution.Globals;
            var argsBuilder = new StringBuilder("deploy");
            AddArgIfExists(g, "dockerapp_applocation", null, argsBuilder);
            AddArgIfExists(g, "dockerapp_orchestrator", "--orchestrator", argsBuilder);
            AddArgIfExists(g, "dockerapp_stackname", "--name", argsBuilder);
            AddArgIfExists(g, "dockerapp_namespace", "--namespace", argsBuilder);
            AddArgIfExists(g, "dockerapp_kubeconfig", "--kubeconfig", argsBuilder);
            var parameters = g.GetOrNull<string>("dockerapp_parameters");

            if (parameters !=null)
            {
                foreach (string s in (parameters).Split('\n')) {
                    argsBuilder.Append($" -s {s}");
                }
            }
            System.Diagnostics.Process proc = new System.Diagnostics.Process();
            proc.StartInfo.FileName = "docker-app";
            proc.StartInfo.UseShellExecute = false;
            proc.StartInfo.RedirectStandardError = true;
            proc.StartInfo.RedirectStandardOutput = true;
            proc.StartInfo.Arguments = argsBuilder.ToString();

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

            generalPane.OutputString("Deploy command: docker-app " + argsBuilder.ToString() + System.Environment.NewLine);
            generalPane.Activate(); // Brings this pane into view
            while (!proc.StandardOutput.EndOfStream)
                generalPane.OutputString(proc.StandardOutput.ReadLine() + System.Environment.NewLine);
            while (!proc.StandardError.EndOfStream)
                generalPane.OutputString(proc.StandardError.ReadLine() + System.Environment.NewLine);
        }
    }
}
