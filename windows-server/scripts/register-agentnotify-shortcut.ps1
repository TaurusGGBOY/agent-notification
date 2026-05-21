param(
    [string]$ExePath = "D:\project\agent-notification\windows-server\agent-notify-server.exe",
    [string]$WorkDir = "D:\project\agent-notification\windows-server",
    [string]$AppId = "AgentNotify"
)

$ErrorActionPreference = "Stop"

$shortcutPath = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\AgentNotify.lnk"
$shortcutDir = Split-Path $shortcutPath -Parent
New-Item -ItemType Directory -Force -Path $shortcutDir | Out-Null

Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
using System.Text;

[ComImport, Guid("00021401-0000-0000-C000-000000000046")]
public class CShellLink { }

[ComImport, InterfaceType(ComInterfaceType.InterfaceIsIUnknown), Guid("000214F9-0000-0000-C000-000000000046")]
public interface IShellLinkW {
    void GetPath([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszFile, int cchMaxPath, IntPtr pfd, uint fFlags);
    void GetIDList(out IntPtr ppidl);
    void SetIDList(IntPtr pidl);
    void GetDescription([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszName, int cchMaxName);
    void SetDescription([MarshalAs(UnmanagedType.LPWStr)] string pszName);
    void GetWorkingDirectory([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszDir, int cchMaxPath);
    void SetWorkingDirectory([MarshalAs(UnmanagedType.LPWStr)] string pszDir);
    void GetArguments([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszArgs, int cchMaxPath);
    void SetArguments([MarshalAs(UnmanagedType.LPWStr)] string pszArgs);
    void GetHotkey(out short pwHotkey);
    void SetHotkey(short wHotkey);
    void GetShowCmd(out int piShowCmd);
    void SetShowCmd(int iShowCmd);
    void GetIconLocation([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszIconPath, int cchIconPath, out int piIcon);
    void SetIconLocation([MarshalAs(UnmanagedType.LPWStr)] string pszIconPath, int iIcon);
    void SetRelativePath([MarshalAs(UnmanagedType.LPWStr)] string pszPathRel, uint dwReserved);
    void Resolve(IntPtr hwnd, uint fFlags);
    void SetPath([MarshalAs(UnmanagedType.LPWStr)] string pszFile);
}

[ComImport, InterfaceType(ComInterfaceType.InterfaceIsIUnknown), Guid("0000010b-0000-0000-C000-000000000046")]
public interface IPersistFile {
    void GetClassID(out Guid pClassID);
    void IsDirty();
    void Load([MarshalAs(UnmanagedType.LPWStr)] string pszFileName, uint dwMode);
    void Save([MarshalAs(UnmanagedType.LPWStr)] string pszFileName, bool fRemember);
    void SaveCompleted([MarshalAs(UnmanagedType.LPWStr)] string pszFileName);
    void GetCurFile([MarshalAs(UnmanagedType.LPWStr)] out string ppszFileName);
}

[StructLayout(LayoutKind.Sequential, Pack = 4)]
public struct PropertyKey {
    public Guid fmtid;
    public uint pid;
    public PropertyKey(Guid fmtid, uint pid) { this.fmtid = fmtid; this.pid = pid; }
}

[StructLayout(LayoutKind.Sequential)]
public struct PropVariant {
    public ushort vt;
    public ushort wReserved1;
    public ushort wReserved2;
    public ushort wReserved3;
    public IntPtr p;
    public int p2;
    public static PropVariant FromString(string value) {
        PropVariant pv = new PropVariant();
        pv.vt = 31;
        pv.p = Marshal.StringToCoTaskMemUni(value);
        return pv;
    }
}

[ComImport, InterfaceType(ComInterfaceType.InterfaceIsIUnknown), Guid("886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99")]
public interface IPropertyStore {
    void GetCount(out uint cProps);
    void GetAt(uint iProp, out PropertyKey pkey);
    void GetValue(ref PropertyKey key, out PropVariant pv);
    void SetValue(ref PropertyKey key, ref PropVariant pv);
    void Commit();
}

public static class ShortcutAppId {
    [DllImport("Ole32.dll")]
    public static extern int PropVariantClear(ref PropVariant pvar);

    public static void Create(string shortcutPath, string exePath, string workDir, string appId) {
        IShellLinkW shellLink = (IShellLinkW)new CShellLink();
        shellLink.SetPath(exePath);
        shellLink.SetWorkingDirectory(workDir);
        shellLink.SetDescription("Agent Notify Server");
        shellLink.SetIconLocation(exePath, 0);

        IPropertyStore store = (IPropertyStore)shellLink;
        PropertyKey appIdKey = new PropertyKey(new Guid("9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3"), 5);
        PropVariant pv = PropVariant.FromString(appId);
        try {
            store.SetValue(ref appIdKey, ref pv);
            store.Commit();
        } finally {
            PropVariantClear(ref pv);
        }

        ((IPersistFile)shellLink).Save(shortcutPath, true);
    }
}
"@

[ShortcutAppId]::Create($shortcutPath, $ExePath, $WorkDir, $AppId)

Get-Item $shortcutPath | Select-Object FullName, Length, LastWriteTime
Write-Host "Registered AppUserModelID: $AppId"
