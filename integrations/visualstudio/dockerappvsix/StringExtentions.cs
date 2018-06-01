using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace dockerappvsix
{
    static class StringExtentions
    {
        public static string QuoteAndStripCarriageReturns(this string s)
        {
            s = s.Replace("\r", "")
                .Replace("\"", "\\\"");
            return $"\"{s}\"";
        }
    }
}
