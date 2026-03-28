using System;
using System.Runtime.InteropServices;
using System.Runtime.InteropServices.WindowsRuntime;
using System.Windows.Forms;
using Windows.Security.Credentials.UI;

namespace WindowsHelloLink
{
    internal static class Program
    {
        private const string AppTitle = "else-toolbox";

        [STAThread]
        private static int Main(string[] args)
        {
            try
            {
                if (args.Length == 0)
                {
                    Console.Error.WriteLine("missing command");
                    return 1;
                }

                if (args[0] == "check")
                {
                    Console.WriteLine(AsyncBridge.Await(UserConsentVerifier.CheckAvailabilityAsync()).ToString());
                    return 0;
                }

                if (args[0] == "verify")
                {
                    if (args.Length < 2)
                    {
                        Console.Error.WriteLine("missing prompt");
                        return 1;
                    }

                    object factory = WindowsRuntimeMarshal.GetActivationFactory(typeof(UserConsentVerifier));
                    var interop = (IUserConsentVerifierInterop)factory;
                    Guid iid = typeof(Windows.Foundation.IAsyncOperation<UserConsentVerificationResult>).GUID;
                    using (var owner = CreateAnchorWindow())
                    {
                        var operation = interop.RequestVerificationForWindowAsync(owner.Handle, args[1], ref iid);
                        var result = AsyncBridge.Await(operation);
                        Console.WriteLine(result.ToString());
                        return 0;
                    }
                }

                Console.Error.WriteLine("unknown command");
                return 1;
            }
            catch (Exception ex)
            {
                Console.Error.WriteLine(ex.Message);
                return 1;
            }
        }

        private static AnchorForm CreateAnchorWindow()
        {
            Rect rect;
            IntPtr appWindow = NativeMethods.FindWindowByTitle(AppTitle);
            if (appWindow != IntPtr.Zero && NativeMethods.TryGetWindowRect(appWindow, out rect))
            {
                return new AnchorForm(rect);
            }

            return new AnchorForm(null);
        }

    }

    internal sealed class AnchorForm : Form
    {
        public AnchorForm(Rect? anchor)
        {
            FormBorderStyle = FormBorderStyle.None;
            ShowInTaskbar = false;
            StartPosition = FormStartPosition.Manual;
            Width = 1;
            Height = 1;
            Opacity = 0;

            if (anchor.HasValue)
            {
                var rect = anchor.Value;
                Location = new System.Drawing.Point(
                    rect.Left + ((rect.Right - rect.Left) / 2),
                    rect.Top + ((rect.Bottom - rect.Top) / 2)
                );
            }
            else
            {
                Location = new System.Drawing.Point(-32000, -32000);
            }

            Show();
            Activate();
        }
    }

    [ComImport]
    [Guid("39E050C3-4E74-441A-8DC0-B81104DF949C")]
    [InterfaceType(ComInterfaceType.InterfaceIsIInspectable)]
    internal interface IUserConsentVerifierInterop
    {
        Windows.Foundation.IAsyncOperation<UserConsentVerificationResult> RequestVerificationForWindowAsync(
            IntPtr appWindow,
            [MarshalAs(UnmanagedType.HString)] string message,
            ref Guid riid);
    }
}
