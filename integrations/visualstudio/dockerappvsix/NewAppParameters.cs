using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace dockerappvsix
{
    public class NewAppParameters
    {
        public string Name { get; set; }
        public string Description { get; set; }
        public string Maintainers { get; set; }
        public bool SingleFile { get; set; }

    }
}
