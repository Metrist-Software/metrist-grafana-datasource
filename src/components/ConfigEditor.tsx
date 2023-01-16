import React, { ChangeEvent, PureComponent } from 'react';
import { Card, InlineFieldRow, LegacyForms } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { DataSourceOptions, SecureJsonData } from '../types';

const { SecretFormField } = LegacyForms;

interface Props extends DataSourcePluginOptionsEditorProps<DataSourceOptions> { }

interface State { }

export class ConfigEditor extends PureComponent<Props, State> {
  onAPIKeyChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonData: {
        apiKey: event.target.value,
      },
    });
  };

  onResetAPIKey = () => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        apiKey: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        apiKey: '',
      },
    });
  };

  render() {
    const { options } = this.props;
    const { secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as SecureJsonData;

    return (
      <div className="gf-form-group">
        <InlineFieldRow>
          <Card>
            <Card.Heading>Getting Started</Card.Heading>
            <Card.Description>
              <ol style={{ paddingLeft: '1em' }}>
                <li>Create a Metrist account <a href="https://app.metrist.io/login/signup">https://app.metrist.io/login/signup</a></li>
                <li>Complete the signup and login to your account</li>
                <li>Head over to the profile page <a href="https://app.metrist.io/profile">https://app.metrist.io/profile</a></li>
                <li>Copy the Auth token by clicking &rdquo;Copy to clipboard&rdquo;</li>
                <li>Paste the Auth token below</li>
              </ol>
            </Card.Description>
            <Card.Figure>
              <img src="https://assets.metrist.io/monitor-logos/metrist.png" alt="Metrist Logo" />
            </Card.Figure>
          </Card>
        </InlineFieldRow>
        <div className="gf-form-inline">
          <div className="gf-form">
            <SecretFormField
              isConfigured={(secureJsonFields && secureJsonFields.apiKey) as boolean}
              value={secureJsonData.apiKey || ''}
              label="Auth token"
              labelWidth={6}
              inputWidth={20}
              onReset={this.onResetAPIKey}
              onChange={this.onAPIKeyChange}
            />
          </div>
        </div>
      </div>
    );
  }
}
