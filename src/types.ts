import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface Query extends DataQuery {
  monitors: string[];
  includeShared: boolean;
}

export const defaultQuery: Partial<Query> = {
  includeShared: true
};

export interface DataSourceOptions extends DataSourceJsonData {
}

export interface SecureJsonData {
  apiKey?: string;
}
