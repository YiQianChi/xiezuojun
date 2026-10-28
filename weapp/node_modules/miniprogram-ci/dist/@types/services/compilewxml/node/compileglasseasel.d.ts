declare const _default: (content: {
    wxmlParserPath: string;
    tempCompileConfigFilePath: string;
    wxmlAndWxsFiles: Record<string, string>;
    glassEaselTemplateCompilerPath: string;
}) => Promise<{
    templateResults: Record<string, string>;
    wxmlDependencies: Record<string, string[]>;
    scriptResult: string;
    runtimeString: string;
    runtimeGlobals: string;
    wxsEnvContent: string;
}>;
export = _default;
