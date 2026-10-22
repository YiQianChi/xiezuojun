@echo off
rem 写作菌 · 发布签名一键脚本（构建未签名包后运行）
rem 用法：DevEco Build 出 entry-default-unsigned.hap 后，双击本脚本
set JAVA=E:\DevEco Studio\jbr\bin\java.exe
set SIGNJAR=E:\DevEco Studio\sdk\default\openharmony\toolchains\lib\hap-sign-tool.jar
set SRC=D:\xiezuojun\harmony\entry\build\default\outputs\default\entry-default-unsigned.hap
set OUT=D:\xiezuojun\harmony\entry\build\default\outputs\default\entry-default-release-signed.hap
set KS=D:\xiezuojun\harmony\signing\xiezuojun.p12
set CER=D:\xiezuojun\harmony\signing\release.cer
set P7B=D:\xiezuojun\harmony\signing\release.p7b
"%JAVA%" -jar "%SIGNJAR%" sign-app -keyAlias xzj -signAlg SHA256withECDSA -mode localSign ^
  -appCertFile "%CER%" -profileFile "%P7B%" -inFile "%SRC%" -keystoreFile "%KS%" ^
  -outFile "%OUT%" -keyPwd Le199789.. -keystorePwd Le199789..
if %errorlevel%==0 (echo.&& echo === 签名成功: %OUT% ===) else (echo.&& echo === 签名失败 ===)
pause
