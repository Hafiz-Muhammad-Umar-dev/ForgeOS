export interface GitFile {
  readonly path: string;
  readonly status: GitFileStatus;
  readonly staged: boolean;
  readonly isUntracked: boolean;
  readonly isIgnored: boolean;
}

export type GitFileStatus = "added" | "modified" | "deleted" | "renamed" | "copied" | "unmerged";

export interface GitBranch {
  readonly name: string;
  readonly isRemote: boolean;
  readonly isCurrent: boolean;
  readonly ahead: number;
  readonly behind: number;
  readonly lastCommit?: string;
  readonly lastCommitTime?: number;
}

export interface GitCommit {
  readonly oid: string;
  readonly message: string;
  readonly author: string;
  readonly timestamp: number;
  readonly branch?: string;
  readonly parentCount: number;
}

export interface GitDiff {
  readonly file: string;
  readonly added: number;
  readonly removed: number;
  readonly content: string;
}

export interface GitStatus {
  readonly currentBranch: string;
  readonly changed: number;
  readonly staged: number;
  readonly untracked: number;
  readonly ignored: number;
  readonly behind: number;
  readonly ahead: number;
}

export interface GitMetrics {
  readonly repositorySize: string;
  readonly commitCount: number;
  readonly branchCount: number;
  readonly changedFiles: number;
  readonly stagedFiles: number;
  readonly lastFetch?: string;
}

export interface MergeConflict {
  readonly file: string;
  readonly status: "conflicted" | "resolved";
}
