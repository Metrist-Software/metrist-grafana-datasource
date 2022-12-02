import defaults from 'lodash/defaults';

import React, { PureComponent } from 'react';
import { InlineField, InlineFieldRow, MultiSelect, Select } from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { DataSource } from '../datasource';
import { defaultQuery, DataSourceOptions, Query } from '../types';

type Props = QueryEditorProps<DataSource, Query, DataSourceOptions>;

export class QueryEditor extends PureComponent<Props> {
  queryTypeChange = (val: SelectableValue<string>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, queryType: val.value as string });
    onRunQuery();
  };

  onMonitorsChange = (vals: Array<SelectableValue<string>>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, monitors: vals.map(v => v.value as string) });
    onRunQuery();
  };

  render() {
    const query = defaults(this.props.query, defaultQuery);
    const { monitors, queryType } = query;

    return (
      <div className="gf-form">
        <InlineFieldRow>
          <InlineField label="Type" labelWidth={14}>
            <Select
              options={[{
                label: 'Errors',
                value: 'GetMonitorErrors'
              },
              {
                label: 'Telemetry',
                value: 'GetMonitorTelemetry'
              },
              {
                label: 'Status Page Changes',
                value: 'GetMonitorStatusPageChanges'
              },
              {
                label: 'Status',
                value: 'GetMonitorStatus'
              }
              ]}
              width={32}
              value={queryType}
              onChange={this.queryTypeChange}
            />
          </InlineField>
          <InlineField label="Monitor" labelWidth={14}>
            <MultiSelect
              options={[{
                label: 'AWS Lambda',
                value: 'awslambda'
              },
              {
                label: 'AWS EKS',
                value: 'awseks'
              },
              {
                label: 'Heroku',
                value: 'heroku'
              }
              ]}
              width={32}
              value={monitors}
              onChange={this.onMonitorsChange}
            />
          </InlineField>
        </InlineFieldRow>
      </div>
    );
  }
}
