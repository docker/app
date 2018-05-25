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
    class SettingsDialog: DialogWindow
    {
        Grid grid;
        RadioButton rKube;
        RadioButton rSwarm;
        TextBox tKubeConfig;
        TextBox tStackName;
        TextBox tNamespace;
        TextBox tSettings;
        Button bOk;
        Button bCancel;
        bool validated;
        private void AddGrid(Grid g, System.Windows.UIElement el, int r, int c)
        {
            Grid.SetRow(el, r);
            Grid.SetColumn(el, c);
            g.Children.Add(el);
        }
        public SettingsDialog()
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
            grid.RowDefinitions.Add(new RowDefinition());
            grid.RowDefinitions.Add(new RowDefinition());
            grid.ColumnDefinitions.Add(new ColumnDefinition());
            grid.ColumnDefinitions.Add(new ColumnDefinition());
            Label l = new Label();
            l.Content = "orchestrator";
            AddGrid(grid, l, 0, 0);
            rKube = new RadioButton();
            rKube.Content = "kubernetes";
            rSwarm = new RadioButton();
            rSwarm.Content = "swarm";
            AddGrid(grid, rSwarm, 1, 0);
            AddGrid(grid, rKube, 1, 1);
            tKubeConfig = new TextBox();
            AddGrid(grid, new Label { Content = "Kube config" }, 2, 0);
            AddGrid(grid, tKubeConfig, 2, 1);
            tStackName = new TextBox();
            AddGrid(grid, new Label { Content = "Stack name" }, 3, 0);
            AddGrid(grid, tStackName, 3, 1);
            tNamespace = new TextBox();
            AddGrid(grid, new Label { Content = "Namespace" }, 4, 0);
            AddGrid(grid, tNamespace, 4, 1);
            tSettings = new TextBox { MinLines = 3, AcceptsReturn = true };
            AddGrid(grid, new Label { Content = "Override settings" }, 5, 0);
            AddGrid(grid, tSettings, 5, 1);
            bOk = new Button { Content = "OK"};
            bCancel = new Button { Content = "Cancel" };
            AddGrid(grid, bOk, 6, 0);
            AddGrid(grid, bCancel, 6, 1);
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
        private void SaveText(Globals g, string name, string text)
        {
            g[name] = text;
            g.set_VariablePersists(name, true);
        }
        public bool Validated()
        {
            return validated;
        }
        public void Save(Globals g)
        {
            string orc = (rKube.IsChecked ?? false) ? "kubernetes" : "swarm";
            SaveText(g, "dockerapp_orchestrator", orc);
            SaveText(g, "dockerapp_kubeconfig", tKubeConfig.Text);
            SaveText(g, "dockerapp_namespace", tNamespace.Text);
            SaveText(g, "dockerapp_stackname", tStackName.Text);
            SaveText(g, "dockerapp_settings", tSettings.Text);
        }
        private void LoadText(Globals g, string key, TextBox target)
        {
            if (g.get_VariableExists(key))
                target.Text = g[key] as string;
        }
        public void Load(Globals g)
        {
            bool isKube = false;
            if (g.get_VariableExists("dockerapp_orchestrator"))
                isKube = g["dockerapp_orchestrator"] as string == "kubernetes";
            rKube.IsChecked = isKube;
            rSwarm.IsChecked = !isKube;
            LoadText(g, "dockerapp_kubeconfig", tKubeConfig);
            LoadText(g, "dockerapp_namespace", tNamespace);
            LoadText(g, "dockerapp_stackname", tStackName);
            LoadText(g, "dockerapp_settings", tSettings);
        }
    }
    /// <summary>
    /// Command handler
    /// </summary>
    internal sealed class CommandSettings
    {
        /// <summary>
        /// Command ID.
        /// </summary>
        public const int CommandId = 4130;

        /// <summary>
        /// Command menu group (command set GUID).
        /// </summary>
        public static readonly Guid CommandSet = new Guid("0113e9de-ef33-4d36-9c72-75012c5afd35");

        /// <summary>
        /// VS Package that provides this command, not null.
        /// </summary>
        private readonly AsyncPackage package;

        /// <summary>
        /// Initializes a new instance of the <see cref="CommandSettings"/> class.
        /// Adds our command handlers for menu (commands must exist in the command table file)
        /// </summary>
        /// <param name="package">Owner package, not null.</param>
        /// <param name="commandService">Command service to add command to, not null.</param>
        private CommandSettings(AsyncPackage package, OleMenuCommandService commandService)
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
        public static CommandSettings Instance
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
            // Verify the current thread is the UI thread - the call to AddCommand in CommandSettings's constructor requires
            // the UI thread.
            ThreadHelper.ThrowIfNotOnUIThread();

            OleMenuCommandService commandService = await package.GetServiceAsync((typeof(IMenuCommandService))) as OleMenuCommandService;
            Instance = new CommandSettings(package, commandService);
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
            SettingsDialog sd = new SettingsDialog();
            sd.Load(g);
            sd.ShowModal();
            if (sd.Validated())
            {
                sd.Save(g);
            }
        }
    }
}
