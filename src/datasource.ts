import { Query, DataSourceOptions } from './types';
import { DataSourceWithBackend } from '@grafana/runtime';
import { DataSourceInstanceSettings } from '@grafana/data';

export class DataSource extends DataSourceWithBackend<Query, DataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<DataSourceOptions>) {
    super(instanceSettings);
  }
}
