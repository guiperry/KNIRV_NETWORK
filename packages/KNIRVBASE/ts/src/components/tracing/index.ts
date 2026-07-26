export interface Span {
  spanId: string;
  traceId: string;
  parentSpanId?: string;
  operationName: string;
  startTime: number;
  endTime?: number;
  attributes: Record<string, any>;
  status: SpanStatus;
}

export enum SpanStatus {
  Unset = 'unset',
  Ok = 'ok',
  Error = 'error',
}

export interface SpanContext {
  traceId: string;
  spanId: string;
}

export interface TracerProvider {
  getTracer(name: string): Tracer;
  shutdown(): Promise<void>;
}

export interface Tracer {
  startSpan(
    name: string,
    options?: {
      parent?: SpanContext;
      attributes?: Record<string, any>;
    }
  ): Span;
}

export interface JaegerExporterConfig {
  endpoint: string;
  agentHost?: string;
  agentPort?: number;
}

class ConsoleSpanExporter implements SpanExporter {
  async export(spans: Span[]): Promise<void> {
    for (const span of spans) {
      console.log(JSON.stringify({
        traceId: span.traceId,
        spanId: span.spanId,
        operationName: span.operationName,
        startTime: new Date(span.startTime).toISOString(),
        endTime: span.endTime ? new Date(span.endTime).toISOString() : null,
        duration: span.endTime ? span.endTime - span.startTime : null,
        attributes: span.attributes,
        status: span.status,
      }));
    }
  }

  async shutdown(): Promise<void> {}
}

interface SpanExporter {
  export(spans: Span[]): Promise<void>;
  shutdown(): Promise<void>;
}

class SimpleSpanProcessor implements SpanProcessor {
  private exporter: SpanExporter;
  private buffer: Span[] = [];
  private maxQueueSize: number = 100;
  private scheduledDelayMS: number = 5000;

  constructor(exporter: SpanExporter) {
    this.exporter = exporter;
    this.startScheduledTask();
  }

  onStart(span: Span): void {}

  onEnd(span: Span): void {
    this.buffer.push(span);
    if (this.buffer.length >= this.maxQueueSize) {
      this.flush();
    }
  }

  shutdown(): Promise<void> {
    return this.flush();
  }

  private async flush(): Promise<void> {
    if (this.buffer.length === 0) return;
    const spans = this.buffer.splice(0);
    try {
      await this.exporter.export(spans);
    } catch (e) {
      console.error('failed to export spans:', e);
    }
  }

  private startScheduledTask(): void {
    setInterval(() => {
      this.flush();
    }, this.scheduledDelayMS);
  }
}

interface SpanProcessor {
  onStart(span: Span): void;
  onEnd(span: Span): void;
  shutdown(): Promise<void>;
}

export async function initTracer(
  serviceName: string,
  jaegerEndpoint: string
): Promise<TracerProvider> {
  const consoleExporter = new ConsoleSpanExporter();
  const spanProcessor = new SimpleSpanProcessor(consoleExporter);

  return new SimpleTracerProvider(serviceName, spanProcessor);
}

export function startSpan(
  ctx: any,
  operationName: string,
  attributes?: Record<string, any>
): Span {
  const traceId = generateTraceId();
  const spanId = generateSpanId();

  const span: Span = {
    spanId,
    traceId,
    operationName,
    startTime: Date.now(),
    attributes: attributes || {},
    status: SpanStatus.Unset,
  };

  return span;
}

export function endSpan(span: Span, status?: SpanStatus): void {
  span.endTime = Date.now();
  if (status) {
    span.status = status;
  }
}

export function setSpanAttribute(span: Span, key: string, value: any): void {
  span.attributes[key] = value;
}

export function setSpanStatus(span: Span, status: SpanStatus, message?: string): void {
  span.status = status;
  if (message) {
    span.attributes['error.message'] = message;
  }
}

class SimpleTracerProvider implements TracerProvider {
  private serviceName: string;
  private processor: SpanProcessor;

  constructor(serviceName: string, processor: SpanProcessor) {
    this.serviceName = serviceName;
    this.processor = processor;
  }

  getTracer(name: string): Tracer {
    return new SimpleTracer(this.serviceName, this.processor);
  }

  async shutdown(): Promise<void> {
    await this.processor.shutdown();
  }
}

class SimpleTracer implements Tracer {
  private serviceName: string;
  private processor: SpanProcessor;

  constructor(serviceName: string, processor: SpanProcessor) {
    this.serviceName = serviceName;
    this.processor = processor;
  }

  startSpan(
    name: string,
    options?: {
      parent?: SpanContext;
      attributes?: Record<string, any>;
    }
  ): Span {
    const span = startSpan(null, name, {
      'service.name': this.serviceName,
      ...options?.attributes,
    });

    if (options?.parent) {
      span.parentSpanId = options.parent.spanId;
    }

    this.processor.onStart(span);
    return span;
  }
}

function generateTraceId(): string {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return Array.from(bytes)
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
}

function generateSpanId(): string {
  const bytes = new Uint8Array(8);
  crypto.getRandomValues(bytes);
  return Array.from(bytes)
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
}

export class TraceContext {
  private static currentContext: Map<string, Span> = new Map();

  static setSpan(traceId: string, span: Span): void {
    this.currentContext.set(traceId, span);
  }

  static getSpan(traceId: string): Span | undefined {
    return this.currentContext.get(traceId);
  }

  static removeSpan(traceId: string): void {
    this.currentContext.delete(traceId);
  }

  static clear(): void {
    this.currentContext.clear();
  }
}

export function withSpan<T>(
  ctx: any,
  operationName: string,
  fn: () => Promise<T>,
  attributes?: Record<string, any>
): Promise<T> {
  const span = startSpan(ctx, operationName, attributes);
  try {
    const result = fn();
    if (result instanceof Promise) {
      return result.then(
        (value) => {
          endSpan(span, SpanStatus.Ok);
          return value;
        },
        (error) => {
          setSpanStatus(span, SpanStatus.Error, error.message);
          endSpan(span, SpanStatus.Error);
          throw error;
        }
      );
    }
    endSpan(span, SpanStatus.Ok);
    return result;
  } catch (error: any) {
    setSpanStatus(span, SpanStatus.Error, error.message);
    endSpan(span, SpanStatus.Error);
    throw error;
  }
}