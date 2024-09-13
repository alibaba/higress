import { log, LogLevelValues } from "@higress/proxy-wasm-assemblyscript-sdk/assembly";

enum LogLevel {
  Trace = 0,
  Debug,
  Info,
  Warn,
  Error,
  Critical,
}

export class Log {
  private pluginName: string;

  constructor(pluginName: string) {
    this.pluginName = pluginName;
  }

  private log(level: LogLevel, msg: string): void {
    let formattedMsg = `[${this.pluginName}] ${msg}`;
    switch (level) {
      case LogLevel.Trace:
        log(LogLevelValues.trace, formattedMsg);
        break;
      case LogLevel.Debug:
        log(LogLevelValues.debug, formattedMsg);
        break;
      case LogLevel.Info:
        log(LogLevelValues.info, formattedMsg);
        break;
      case LogLevel.Warn:
        log(LogLevelValues.warn, formattedMsg);
        break;
      case LogLevel.Error:
        log(LogLevelValues.error, formattedMsg);
        break;
      case LogLevel.Critical:
        log(LogLevelValues.critical, formattedMsg);
        break;
    }
  }

  public Trace(msg: string): void {
    this.log(LogLevel.Trace, msg);
  }

  public Debug(msg: string): void {
    this.log(LogLevel.Debug, msg);
  }

  public Info(msg: string): void {
    this.log(LogLevel.Info, msg);
  }

  public Warn(msg: string): void {
    this.log(LogLevel.Warn, msg);
  }

  public Error(msg: string): void {
    this.log(LogLevel.Error, msg);
  }

  public Critical(msg: string): void {
    this.log(LogLevel.Critical, msg);
  }
}