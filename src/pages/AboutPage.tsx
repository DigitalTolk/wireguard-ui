import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api-client";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import type { AppInfo, GitHubRelease, GitHubContributor } from "@/lib/types";

const REPO_OWNER = "DigitalTolk";
const REPO_NAME = "wireguard-ui";
const ORIGINAL_REPO = "ngoduykhanh/wireguard-ui";

export function AboutPage() {
  const { data: appInfo } = useQuery({
    queryKey: ["app-info"],
    queryFn: () => apiGet<AppInfo>("/auth/info"),
  });

  const { data: latestRelease } = useQuery({
    queryKey: ["github-latest-release"],
    queryFn: async () => {
      const res = await fetch(
        `https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest`
      );
      if (!res.ok) return null;
      return res.json() as Promise<GitHubRelease>;
    },
    staleTime: 5 * 60 * 1000,
    retry: false,
  });

  const { data: contributors } = useQuery({
    queryKey: ["github-contributors"],
    queryFn: async () => {
      const res = await fetch(
        `https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/contributors`
      );
      if (!res.ok) return [];
      return res.json() as Promise<GitHubContributor[]>;
    },
    staleTime: 5 * 60 * 1000,
    retry: false,
  });

  const version = appInfo?.app_version ?? "unknown";
  const commit = appInfo?.git_commit ?? "unknown";
  const isOutdated =
    latestRelease?.tag_name &&
    version !== "development" &&
    latestRelease.tag_name !== version &&
    latestRelease.tag_name !== `v${version}`;

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold tracking-tight">About</h2>

      <Card>
        <CardHeader>
          <CardTitle>WireGuard UI</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-5">
          <div className="grid gap-2">
            <Label>Current Version</Label>
            <div className="flex items-center gap-2">
              <Input value={version} readOnly />
              {isOutdated && (
                <Badge variant="destructive">Update available</Badge>
              )}
            </div>
          </div>

          <div className="grid gap-2">
            <Label>Git Commit</Label>
            <Input value={commit} readOnly />
          </div>

          {latestRelease && (
            <>
              <div className="grid gap-2">
                <Label>Latest Release</Label>
                <Input value={latestRelease.tag_name} readOnly />
              </div>
              <div className="grid gap-2">
                <Label>Latest Release Date</Label>
                <Input
                  value={
                    latestRelease.published_at
                      ? new Date(latestRelease.published_at).toLocaleDateString()
                      : "N/A"
                  }
                  readOnly
                />
              </div>
            </>
          )}

          {!latestRelease && (
            <div className="grid gap-2">
              <Label>Latest Release</Label>
              <Skeleton className="h-10 w-full" />
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Contributors</CardTitle>
        </CardHeader>
        <CardContent>
          {contributors && contributors.length > 0 ? (
            <div className="flex flex-wrap gap-3">
              {contributors.map((c) => (
                <a
                  key={c.login}
                  href={c.html_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  title={`${c.login} (${c.contributions} contributions)`}
                  className="group"
                >
                  <img
                    src={c.avatar_url}
                    alt={c.login}
                    className="h-12 w-12 rounded-full border border-border transition-transform group-hover:scale-110"
                  />
                </a>
              ))}
            </div>
          ) : (
            <Skeleton className="h-12 w-48" />
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Project</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p>
            <a
              href={`https://github.com/${REPO_OWNER}/${REPO_NAME}`}
              className="font-medium text-primary underline"
              target="_blank"
              rel="noopener noreferrer"
            >
              {REPO_OWNER}/{REPO_NAME}
            </a>
          </p>

          <Separator />

          <p className="text-muted-foreground">
            Fork of{" "}
            <a
              href={`https://github.com/${ORIGINAL_REPO}`}
              className="text-primary underline"
              target="_blank"
              rel="noopener noreferrer"
            >
              {ORIGINAL_REPO}
            </a>{" "}
            by{" "}
            <a
              href="https://github.com/ngoduykhanh"
              className="text-primary underline"
              target="_blank"
              rel="noopener noreferrer"
            >
              Khanh Ngo
            </a>
          </p>

          <p className="text-muted-foreground">
            Copyright &copy; {new Date().getFullYear()}{" "}
            <a
              href={`https://github.com/${REPO_OWNER}/${REPO_NAME}`}
              className="text-primary underline"
              target="_blank"
              rel="noopener noreferrer"
            >
              WireGuard UI
            </a>
            . All rights reserved.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
