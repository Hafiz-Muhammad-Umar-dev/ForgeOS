export type DeployStatus = "pending" | "deploying" | "building" | "uploading" | "running" | "healthy" | "stopped" | "failed" | "cancelled";

export interface Deployment {
  readonly id: string;
  readonly projectName: string;
  readonly status: DeployStatus;
  readonly branch?: string;
  readonly commit?: string;
  readonly commitMessage?: string;
  readonly image?: string;
  readonly region?: string;
  readonly url?: string;
  readonly createdBy?: string;
  readonly createdAt: string;
  readonly startedAt?: string;
  readonly finishedAt?: string;
  readonly cpu?: number;
  readonly memory?: number;
  readonly network?: number;
  readonly healthStatus?: "healthy" | "degraded" | "unhealthy";
}

export interface DeploymentLog {
  readonly timestamp: string;
  readonly level: "info" | "warn" | "error" | "debug";
  readonly message: string;
  readonly source?: string;
}

export interface DeploymentMetric {
  readonly timestamp: string;
  readonly cpu: number;
  readonly memory: number;
  readonly disk: number;
  readonly network: number;
  readonly requestsPerSec: number;
  readonly latency: number;
  readonly errors: number;
}

export interface EnvVariable {
  readonly key: string;
  readonly value: string;
  readonly isSecret: boolean;
}

export interface HealthCheck {
  readonly type: "http" | "tcp" | "custom";
  readonly endpoint: string;
  readonly status: "healthy" | "degraded" | "unhealthy";
  readonly lastChecked: string;
  readonly interval: number;
}

export interface TimelineStage {
  readonly name: string;
  readonly status: DeployStatus;
  readonly startedAt?: string;
  readonly finishedAt?: string;
  readonly duration?: number;
}
