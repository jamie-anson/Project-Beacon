import { runnerFetch } from '../http.js';

export function getHealth() {
  return runnerFetch('/health').then((data) => {
    if (data && data.services) {
      const healthData = {};
      data.services.forEach((service) => {
        healthData[service.name] = service.status;
      });
      healthData.overall = data.status;
      return healthData;
    }
    return data;
  });
}
