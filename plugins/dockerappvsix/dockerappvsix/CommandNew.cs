using System;
using System.ComponentModel.Design;
using System.Globalization;
using System.IO;
using System.Linq;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using System.Windows.Controls;
using EnvDTE;
using EnvDTE80;
using Microsoft.VisualStudio.PlatformUI;
using Microsoft.VisualStudio.Shell;
using Microsoft.VisualStudio.Shell.Interop;
using Task = System.Threading.Tasks.Task;

namespace dockerappvsix
{       
    internal sealed class CommandNew
    {
        public const int CommandId = 4132;
        
        public static readonly Guid CommandSet = new Guid("0113e9de-ef33-4d36-9c72-75012c5afd35");
        
        private readonly AsyncPackage _package;
        
        private CommandNew(AsyncPackage package, OleMenuCommandService commandService)
        {
            _package = package ?? throw new ArgumentNullException(nameof(package));
            commandService = commandService ?? throw new ArgumentNullException(nameof(commandService));

            var menuCommandID = new CommandID(CommandSet, CommandId);
            var menuItem = new MenuCommand(this.ExecuteAsync, menuCommandID);
            commandService.AddCommand(menuItem);
        }
        
        public static CommandNew Instance
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
            Instance = new CommandNew(package, commandService);
        }
        
        private async void ExecuteAsync(object sender, EventArgs e)
        {
            ThreadHelper.ThrowIfNotOnUIThread();
            NewAppDialog ns = new NewAppDialog();
            if (!(ns.ShowDialog() ?? false)) {
                return;
            }
            var s = ns.Settings;
            var argsBuilder = new StringBuilder("init " + s.Name.QuoteAndStripCarriageReturns());
            if (s.Description != "")
                argsBuilder.Append($" --description {s.Description.QuoteAndStripCarriageReturns()}");
            if (s.Maintainers != "")
            {
                foreach (string m in s.Maintainers?.Split('\n')??Enumerable.Empty<string>()) {
                    argsBuilder.Append($" --maintainer {m.QuoteAndStripCarriageReturns()}");
                }
            }
            if (s.SingleFile)
                argsBuilder.Append(" -s");
            System.Diagnostics.Process proc = new System.Diagnostics.Process();
            proc.StartInfo.FileName = "docker-app";
            proc.StartInfo.UseShellExecute = false;
            proc.StartInfo.RedirectStandardError = true;
            proc.StartInfo.RedirectStandardOutput = true;
            proc.StartInfo.Arguments = argsBuilder.ToString();
            DTE dte = await this._package.GetServiceAsync(typeof(DTE)) as DTE;
            var solutionDir = "";
            if (dte.Solution.FileName != "")
            {
                solutionDir = Path.GetDirectoryName(dte.Solution.FileName);
                proc.StartInfo.WorkingDirectory = solutionDir;
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
                this._package,
                message,
                title,
                OLEMSGICON.OLEMSGICON_INFO,
                OLEMSGBUTTON.OLEMSGBUTTON_OK,
                OLEMSGDEFBUTTON.OLEMSGDEFBUTTON_FIRST);
            var sol2 = (EnvDTE80.Solution2)dte.Solution;

            var solutionItems = FindDockerFolderProject(sol2);
            var path = Path.Combine(solutionDir, s.Name + ".dockerapp");
            if (Directory.Exists(path)) {
                var sf = (SolutionFolder)solutionItems.Object;
                var f = sf.AddSolutionFolder(s.Name+".dockerapp");
                foreach(var file in Directory.GetFiles(path)) {
                    f.ProjectItems.AddFromFile(file);
                }
            } else if (File.Exists(path)) {
                solutionItems.ProjectItems.AddFromFile(path);
            }
        }
        Project FindDockerFolderProject(EnvDTE80.Solution2 s)
        {
            foreach(Project p in s.Projects) {
                if (p.Globals.VariableExists["docker-app-solution-folder"]) {
                    return p;
                }
            }
            var proj = s.AddSolutionFolder("Docker");
            proj.Globals["docker-app-solution-folder"] = true;
            proj.Globals.VariablePersists["docker-app-solution-folder"] = true;
            return proj;
        }
    }
}
