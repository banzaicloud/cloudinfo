export interface Provider {
  provider: string;
  name?: string;
  services: Array<{ service: string }>;
}
