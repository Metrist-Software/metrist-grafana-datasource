import defaults from 'lodash/defaults';

import React, { ChangeEvent, useEffect, useState } from 'react';
import { InlineField, InlineFieldRow, InlineSwitch, LoadingPlaceholder, MultiSelect, Select } from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { DataSource } from '../datasource';
import { defaultQuery, DataSourceOptions, Query } from '../types';

type Props = QueryEditorProps<DataSource, Query, DataSourceOptions>;

export const QueryEditor = (props: Props) => {
  const [monitorSelect, setMonitors] = useState<Array<SelectableValue<string>>>();

  useEffect(() => {
    const dataFetch = async () => {
      try {
        const monitors = await props.datasource.getResource('Monitors');
        setMonitors(monitors);
      } catch (e) {
        console.error(e)
        setMonitors([]);
      }
    };

    dataFetch();
  }, [props.datasource]);

  const queryTypeChange = (val: SelectableValue<string>) => {
    const { onChange, query, onRunQuery } = props;
    onChange({ ...query, queryType: val.value as string });
    onRunQuery();
  };

  const onMonitorsChange = (vals: Array<SelectableValue<string>>) => {
    const { onChange, query, onRunQuery } = props;
    onChange({ ...query, monitors: vals.map(v => v.value as string) });
    onRunQuery();
  };

  const onSharedDataChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = props;
    onChange({ ...query, includeShared: event.currentTarget.checked });
    onRunQuery();
  };

  const additionalFormFields = (queryType: string | undefined) => {
    const query = defaults(props.query, defaultQuery);
    switch (queryType) {
      case 'GetMonitorErrors':
      case 'GetMonitorTelemetry':
        return (
          <InlineField label="Include Shared Data">
            <InlineSwitch
              value={query.includeShared}
              onChange={onSharedDataChange}
            />
          </InlineField>
        )
      case 'GetMonitorStatusPageChanges':
      case 'GetMonitorStatus':
      default:
        return <></>
    }
  }

  const query = defaults(props.query, defaultQuery);
  const { monitors, queryType } = query;

  if (!monitorSelect) {
    return <LoadingPlaceholder text={"Loading.."}></LoadingPlaceholder>
  }

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
            onChange={queryTypeChange}
          />
        </InlineField>
        <InlineField label="Monitor" labelWidth={14}>
          <MultiSelect
            options={monitorSelect}
            width={32}
            value={monitors}
            onChange={onMonitorsChange}
          />
        </InlineField>
        {additionalFormFields(queryType)}
      </InlineFieldRow>
    </div>
  );
}
