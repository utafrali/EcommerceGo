import { Metadata } from 'next';
import SettingsClient from './SettingsClient';

export const metadata: Metadata = {
  title: 'Settings | EcommerceGo',
  description: 'Manage your account settings and password',
};

export default function SettingsPage() {
  return <SettingsClient />;
}
