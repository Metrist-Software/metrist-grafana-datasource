import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface Query extends DataQuery {
  queryText?: string;
  constant: number;
}

export const defaultQuery: Partial<Query> = {
  constant: 1,
};

/**
 * These are options configured for each DataSource instance
 */
export interface DataSourceOptions extends DataSourceJsonData {
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface SecureJsonData {
  apiKey?: string;
}
