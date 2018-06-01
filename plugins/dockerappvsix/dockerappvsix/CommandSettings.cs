using System;
using System.ComponentModel.Design;
using System.Globalization;
using System.Threading;
using System.Threading.Tasks;
using Microsoft.VisualStudio.Shell;
using Microsoft.VisualStudio.Shell.Interop;
using Task = System.Threading.Tasks.Task;
using Microsoft.VisualStudio.PlatformUI;
using System.Windows.Controls;
using EnvDTE;

namespace dockerappvsix
{
    

    internal sealed class CommandSettings
    {

        public const int CommandId = 4130;


        public static readonly Guid CommandSet = new Guid("0113e9de-ef33-4d36-9c72-75012c5afd35");


        private readonly AsyncPackage _package;
        
        private CommandSettings(AsyncPackage package, OleMenuCommandService commandService)
        {
            _package = package ?? throw new ArgumentNullException(nameof(package));
            commandService = commandService ?? throw new ArgumentNullException(nameof(commandService));

            var menuCommandID = new CommandID(CommandSet, CommandId);
            var menuItem = new MenuCommand(this.ExecuteAsync, menuCommandID);
            commandService.AddCommand(menuItem);
        }
        
        public static CommandSettings Instance
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
            // Verify the current thread is the UI thread - the call to AddCommand in CommandSettings's constructor requires
            // the UI thread.
            ThreadHelper.ThrowIfNotOnUIThread();

            OleMenuCommandService commandService = await package.GetServiceAsync((typeof(IMenuCommandService))) as OleMenuCommandService;
            Instance = new CommandSettings(package, commandService);
        }
        
        private async void ExecuteAsync(object sender, EventArgs e)
        {
            ThreadHelper.ThrowIfNotOnUIThread();
            DTE dte = await this._package.GetServiceAsync(typeof(DTE)) as DTE;
            Globals g = dte.Solution.Globals;
            SettingsDialog sd = new SettingsDialog();
            sd.Settings.LoadFromSolution(g);
            if (sd.ShowDialog() ?? false) {
                sd.Settings.Save(g);
            }
        }
    }
}
