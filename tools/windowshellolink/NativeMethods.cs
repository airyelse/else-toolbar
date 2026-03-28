using System;
using System.Runtime.InteropServices;

namespace WindowsHelloLink
{
    [StructLayout(LayoutKind.Sequential)]
    internal struct Rect
    {
        public int Left;
        public int Top;
        public int Right;
        public int Bottom;
    }

    internal static class NativeMethods
    {
        public static IntPtr FindWindowByTitle(string title)
        {
            return FindWindow(null, title);
        }

        public static bool TryGetWindowRect(IntPtr hwnd, out Rect rect)
        {
            return GetWindowRect(hwnd, out rect);
        }

        [DllImport("user32.dll", CharSet = CharSet.Unicode, EntryPoint = "FindWindowW")]
        private static extern IntPtr FindWindow(string lpClassName, string lpWindowName);

        [DllImport("user32.dll")]
        private static extern bool GetWindowRect(IntPtr hWnd, out Rect lpRect);
    }
}
