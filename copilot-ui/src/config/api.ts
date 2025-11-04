const BACKEND_URL = (window as any).RUNTIME_CONFIG?.BACKEND_URL || '/api';

const API_ENDPOINTS = {
  REPOS: `${BACKEND_URL}/v1/repos`,
  IMPORTS: `${BACKEND_URL}/v1/imports`,
  PLAN: (repoId: string) => `${BACKEND_URL}/v1/repos/${repoId}/plan`,
  TOGGLES: (repoId: string, service: string, env: string) => 
    `${BACKEND_URL}/v1/repos/${repoId}/services/${service}/toggles/${env}`,
};

export default API_ENDPOINTS;
