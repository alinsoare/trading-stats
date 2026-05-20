; Inno Setup script for Trading Stats
;
; Build requirements (build machine):
;   - Inno Setup 6+  (https://jrsoftware.org/isinfo.php  or  choco install innosetup)
;   - Two local wheels already built in dist\wheels\ by build_installer.ps1
;
; Compile:
;   iscc TradingStats.iss /DAppVersion=0.2.0
;
; Output: dist\TradingStats-<version>-setup.exe
;
; Target machine requirements:
;   - Python 3.10+  (https://www.python.org/downloads/)
;   - Internet access (pip downloads PySide6, polars, matplotlib from PyPI at install time)

#ifndef AppVersion
  #define AppVersion "0.1.0"
#endif

#define AppName        "Trading Stats"
#define AppPublisher   "local build"
#define AppURL         "https://github.com"
#define AppExeName     "trading-stats.bat"

[Setup]
AppId={{A1B2C3D4-E5F6-7890-ABCD-EF1234567890}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL={#AppURL}
AppSupportURL={#AppURL}
AppUpdatesURL={#AppURL}
; User-level install — no UAC prompt required
DefaultDirName={localappdata}\TradingStats
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
; No admin rights required
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=commandline
OutputDir=dist
OutputBaseFilename=TradingStats-{#AppVersion}-setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
; Show a progress page during the pip install (can be slow)
ShowComponentSizes=no

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Files]
; Copy local app wheels — pip resolves and downloads all third-party deps at install time
Source: "dist\wheels\*.whl"; DestDir: "{app}\wheels"; Flags: ignoreversion

[Icons]
; Start Menu shortcut — pythonw.exe suppresses the console window
Name: "{group}\{#AppName}"; Filename: "{app}\venv\Scripts\pythonw.exe"; Parameters: "-m trading_stats_desktop"; WorkingDir: "{app}"; Comment: "MT5 multi-account trading statistics"

[Run]
; All commands run after files are copied, shown in the "Installing" wizard page
Filename: "python"; Parameters: "-m venv ""{app}\venv"""; \
  StatusMsg: "Creating Python virtual environment..."; \
  Flags: runhidden waituntilterminated
Filename: "{app}\venv\Scripts\pip.exe"; Parameters: "install --upgrade pip --quiet"; \
  StatusMsg: "Upgrading pip..."; \
  Flags: runhidden waituntilterminated
Filename: "{app}\venv\Scripts\pip.exe"; \
  Parameters: "install ""{app}\wheels\trading_stats-*.whl"" ""{app}\wheels\trading_stats_desktop-*.whl"" --quiet"; \
  StatusMsg: "Installing dependencies from PyPI (this may take a few minutes)..."; \
  Flags: runhidden waituntilterminated

[UninstallRun]
; On uninstall, deactivate / remove the venv before the directory is deleted
Filename: "cmd.exe"; Parameters: "/c rmdir /s /q ""{app}\venv"""; \
  Flags: runhidden waituntilterminated; RunOnceId: "RemoveVenv"

[UninstallDelete]
; Remove the entire install directory (wheels + venv)
Type: filesandordirs; Name: "{app}"

[Code]
// ── Python 3.10+ detection ───────────────────────────────────────────────────
// Runs before the installer wizard advances past the first page.

function GetPythonVersion(var Major, Minor: Integer): Boolean;
var
  Output: AnsiString;
  TmpFile: String;
  Cmd: String;
begin
  Result := False;
  TmpFile := ExpandConstant('{tmp}\pyver.txt');
  Cmd := 'python -c "import sys; open(r''' + TmpFile + ''', ''w'').write(str(sys.version_info.major) + ''.'' + str(sys.version_info.minor))"';
  if Exec(ExpandConstant('{cmd}'), '/c ' + Cmd, '', SW_HIDE, ewWaitUntilTerminated, Major) then
  begin
    if LoadStringFromFile(TmpFile, Output) then
    begin
      Major := StrToIntDef(Copy(string(Output), 1, Pos('.', string(Output)) - 1), 0);
      Minor := StrToIntDef(Copy(string(Output), Pos('.', string(Output)) + 1, 10), 0);
      Result := True;
    end;
  end;
end;

function InitializeSetup(): Boolean;
var
  Major, Minor: Integer;
  Msg: String;
begin
  Result := True;
  if not GetPythonVersion(Major, Minor) then
  begin
    Msg := 'Python was not found on your PATH.' + #13#10 + #13#10 +
           'Trading Stats requires Python 3.10 or newer.' + #13#10 +
           'Download it from: https://www.python.org/downloads/' + #13#10 + #13#10 +
           'Make sure to check "Add Python to PATH" during installation.';
    MsgBox(Msg, mbError, MB_OK);
    Result := False;
    Exit;
  end;
  if (Major < 3) or ((Major = 3) and (Minor < 10)) then
  begin
    Msg := Format('Python %d.%d was found, but Trading Stats requires Python 3.10 or newer.', [Major, Minor]) + #13#10 + #13#10 +
           'Download a newer version from: https://www.python.org/downloads/' + #13#10 + #13#10 +
           'Make sure to check "Add Python to PATH" during installation.';
    MsgBox(Msg, mbError, MB_OK);
    Result := False;
  end;
end;
