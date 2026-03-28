using System.Linq;
using System.Reflection;
using System.Threading.Tasks;
using Windows.Foundation;

namespace WindowsHelloLink
{
    internal static class AsyncBridge
    {
        private static readonly MethodInfo AsTaskMethod = typeof(System.WindowsRuntimeSystemExtensions)
            .GetMethods()
            .First(method =>
                method.Name == "AsTask" &&
                method.IsGenericMethodDefinition &&
                method.GetParameters().Length == 1 &&
                method.GetParameters()[0].ParameterType.Name == "IAsyncOperation`1");

        public static TResult Await<TResult>(IAsyncOperation<TResult> operation)
        {
            var task = (Task<TResult>)AsTaskMethod.MakeGenericMethod(typeof(TResult)).Invoke(null, new object[] { operation });
            return task.GetAwaiter().GetResult();
        }
    }
}
