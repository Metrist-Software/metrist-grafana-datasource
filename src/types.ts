import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface Query extends DataQuery {
  monitors: string[];
  checks: string[];
  instances: string[];
  includeShared: boolean;
  fromAlerting: boolean;
}

export const defaultQuery: Partial<Query> = {
  includeShared: true
};

export interface DataSourceOptions extends DataSourceJsonData {
}

export interface SecureJsonData {
  apiKey?: string;
}
