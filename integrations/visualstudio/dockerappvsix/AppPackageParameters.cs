using EnvDTE;
using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Linq;
using System.Runtime.CompilerServices;
using System.Text;
using System.Threading.Tasks;

namespace dockerappvsix
{
    public class AppPackageParameters : INotifyPropertyChanged
    {
        private bool _isSwarm;
        private bool _isKubernetes;
        private string _kubeConfig;
        private string _namespace;
        private string _stackName;
        private string _parameters;

        public bool IsSwarm
        {
            get => _isSwarm;
            set
            {
                _isSwarm = value;
                OnPropertyChanged();
            }
        }
        public bool IsKubernetes
        {
            get => _isKubernetes;
            set
            {
                _isKubernetes = value;
                OnPropertyChanged();
            }
        }
        public string KubeConfig
        {
            get => _kubeConfig;
            set
            {
                _kubeConfig = value;
                OnPropertyChanged();
            }
        }
        public string Namespace
        {
            get => _namespace;
            set
            {
                _namespace = value;
                OnPropertyChanged();
            }
        }
        public string StackName
        {
            get => _stackName;
            set
            {
                _stackName = value;
                OnPropertyChanged();
            }
        }
        public string Parameters
        {
            get => _parameters;
            set
            {
                _parameters = value;
                OnPropertyChanged();
            }
        }

        public void LoadFromSolution(Globals g)
        {
            var orchestrator = g.GetOrNull<string>("dockerapp_orchestrator");
            IsSwarm = orchestrator != "kubernetes";
            IsKubernetes = orchestrator == "kubernetes";
            KubeConfig = g.GetOrNull<string>("dockerapp_kubeconfig");
            Namespace = g.GetOrNull<string>("dockerapp_namespace");
            StackName = g.GetOrNull<string>("dockerapp_stackname");
            Parameters = g.GetOrNull<string>("dockerapp_parameters");
        }

        public void Save(Globals g)
        {
            var orchestrator = IsKubernetes ? "kubernetes" : "swarm";
            g["dockerapp_orchestrator"] = orchestrator;
            g.VariablePersists["dockerapp_orchestrator"] = true;
            g["dockerapp_kubeconfig"] = KubeConfig;
            g.VariablePersists["dockerapp_kubeconfig"] = true;
            g["dockerapp_namespace"] = Namespace;
            g.VariablePersists["dockerapp_namespace"] = true;
            g["dockerapp_stackname"] = StackName;
            g.VariablePersists["dockerapp_stackname"] = true;
            g["dockerapp_parameters"] = Parameters;
            g.VariablePersists["dockerapp_parameters"] = true;
        }

        private void OnPropertyChanged([CallerMemberName]string name = null)
        {
            PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(name));
        }
        public event PropertyChangedEventHandler PropertyChanged;
    }
}
