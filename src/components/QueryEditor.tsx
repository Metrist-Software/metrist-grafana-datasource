import defaults from 'lodash/defaults';

import React, { ChangeEvent, PureComponent } from 'react';
import { InlineField, InlineFieldRow, InlineSwitch, MultiSelect, Select } from '@grafana/ui';
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

  onSharedDataChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, includeShared: event.currentTarget.checked });
    onRunQuery();
  };

  additionalFormFields = (queryType: string | undefined) => {
    const query = defaults(this.props.query, defaultQuery);
    switch (queryType) {
      case 'GetMonitorErrors':
      case 'GetMonitorTelemetry':
        return (
          <InlineField label="Include Shared Data">
            <InlineSwitch
              value={query.includeShared}
              onChange={this.onSharedDataChange}
            />
          </InlineField>
        )
      case 'GetMonitorStatusPageChanges':
      case 'GetMonitorStatus':
      default:
        return <></>
    }
  }

  render() {
    const query = defaults(this.props.query, defaultQuery);
    const { monitors, queryType } = query;

    return (
      <div style={{ width: '100%' }}>
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
          {this.additionalFormFields(queryType)}
        </InlineFieldRow>
      </div>
    );
  }
}
