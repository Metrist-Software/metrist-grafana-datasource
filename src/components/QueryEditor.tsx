import defaults from 'lodash/defaults';

import React, { ChangeEvent, useEffect, useState } from 'react';
import { InlineField, InlineFieldRow, InlineLabel, InlineSwitch, LoadingPlaceholder, MultiSelect, Select } from '@grafana/ui';
import { QueryEditorProps, SelectableValue, CoreApp } from '@grafana/data';
import { DataSource } from '../datasource';
import { defaultQuery, DataSourceOptions, Query } from '../types';

type Props = QueryEditorProps<DataSource, Query, DataSourceOptions>;

export const QueryEditor = (props: Props) => {
  const [monitorSelect, setMonitors] = useState<Array<SelectableValue<string>>>();
  const [checkSelect, setChecks] = useState<Array<SelectableValue<string>>>();
  const [instanceSelect, setInstances] = useState<Array<SelectableValue<string>>>();
  const [buildHash, setBuildHash] = useState<string>();

  // On load set the fromAlerting query var to true if CloudAlerting or UnifiedAlerting
  useEffect(()=>{
    const { app, onChange, query } = props;
    switch (app) {
      case CoreApp.CloudAlerting:
      case CoreApp.UnifiedAlerting:
        onChange({ ...query, fromAlerting: true });
        break;
    }   
  }, [])

  // Set the initial monitor list and hash
  useEffect(() => {
    const dataFetch = async () => {
      try {
        const monitors = await props.datasource.getResource('Monitors');
        const hash = (await props.datasource.getResource('BuildHash')).hash;
        setBuildHash(hash);
        setMonitors(monitors);
      } catch (e) {
        console.error(e)
        setMonitors([]);
        setBuildHash("");
      }
    };

    dataFetch();
  }, [props.datasource]);

  // If query.monitors or query.includeShared change, then reload the checks and instances list
  useEffect(() => {
    const dataFetch = async () => {
      try {
        if (props.query.monitors != null) {
          const checks = await props.datasource.getResource('Checks', { monitors: props.query.monitors, includeShared: props.query.includeShared });
          setChecks(checks)
    
          const instances = await props.datasource.getResource('Instances', { monitors: props.query.monitors, includeShared: props.query.includeShared });
          setInstances(instances)
        }
      } catch (e) {
        console.error(e)
        setChecks([]);
        setInstances([]);
      }
    };

    dataFetch();
  }, [props.datasource, props.query.monitors, props.query.includeShared]);

  const queryTypeChange = (val: SelectableValue<string>) => {
    const { onChange, query, onRunQuery } = props;
    onChange({ ...query, queryType: val.value as string });
    onRunQuery();
  };

  const onMonitorsChange = async (vals: Array<SelectableValue<string>>) => {  
    const { onChange, query, onRunQuery } = props;

    const monitors = vals.map(v => v.value as string)

    onChange({ ...query, monitors: monitors });
    onRunQuery();
  };

  const onSharedDataChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = props;
    onChange({ ...query, includeShared: event.currentTarget.checked });
    onRunQuery();
  };

  const onChecksChange = (vals: Array<SelectableValue<string>>) => {
    const { onChange, query, onRunQuery } = props;
    onChange({ ...query, checks: vals.map(v => v.value as string) });
    onRunQuery();
  };

  const onInstancesChange = (vals: Array<SelectableValue<string>>) => {
    const { onChange, query, onRunQuery } = props;
    onChange({ ...query, instances: vals.map(v => v.value as string) });
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

  const additionalFormRows = (queryType: string | undefined) => {
    const query = defaults(props.query, defaultQuery);
    switch (queryType) {
      case 'GetMonitorErrors':
      case 'GetMonitorTelemetry':
        return (
            <div>
            <InlineLabel>Additional Filters (Optional)</InlineLabel>
            <InlineFieldRow>            
            <InlineField label="Checks" labelWidth={14}>
              <MultiSelect
                options={checkSelect}
                width={32}
                value={query.checks}
                onChange={onChecksChange}
              />
            </InlineField>
            <InlineField label="Instances" labelWidth={14}>
              <MultiSelect
                options={instanceSelect}
                width={32}
                value={query.instances}
                onChange={onInstancesChange}
              />
            </InlineField>
          </InlineFieldRow>
          </div>
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

  if (buildHash === "") {
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
      {additionalFormRows(queryType)}
     <div><sub>Query Version: {buildHash}</sub></div>
    </div>
  );
}
