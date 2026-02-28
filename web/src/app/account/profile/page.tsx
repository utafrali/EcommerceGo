import { Metadata } from 'next';
import ProfileClient from './ProfileClient';

export const metadata: Metadata = {
  title: 'Profile | EcommerceGo',
  description: 'View and edit your profile information',
};

export default function ProfilePage() {
  return <ProfileClient />;
}
