using EnvDTE;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace dockerappvsix
{
    static class GlobalsExtensions
    {
        public static T GetOrNull<T>(this Globals g, string key) where T : class
        {
            if (!g.VariableExists[key]) {
                return null;
            }
            return g[key] as T;
        }
    }
}
