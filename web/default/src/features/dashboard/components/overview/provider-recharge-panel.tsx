/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { ExternalLink, Wallet } from 'lucide-react'
import { useTranslation } from 'react-i18next'

const PROVIDER_RECHARGE_LINKS = [
  {
    name: '阿里云百炼',
    url: 'https://usercenter2.aliyun.com/finance/fund-management/recharge',
  },
  {
    name: '火山引擎',
    url: 'https://console.volcengine.com/finance/recharge',
  },
  { name: '腾讯云', url: 'https://buy.cloud.tencent.com/recharge' },
  { name: 'MiniMax', url: 'https://platform.minimaxi.com/' },
  { name: 'Agnes', url: 'https://apihub.agnes-ai.com/' },
  { name: 'xAI', url: 'https://console.x.ai/' },
  { name: 'Google', url: 'https://console.cloud.google.com/billing' },
  { name: 'DeepSeek', url: 'https://platform.deepseek.com/top_up' },
]

export function ProviderRechargePanel() {
  const { t } = useTranslation()

  return (
    <div className='bg-card rounded-2xl border p-4 shadow-xs sm:p-5'>
      <div className='flex flex-col gap-1'>
        <div className='flex items-center gap-2'>
          <Wallet className='text-muted-foreground size-4' aria-hidden='true' />
          <h3 className='text-base font-semibold'>{t('Provider recharge')}</h3>
        </div>
        <p className='text-muted-foreground text-sm'>
          {t('Open the official recharge page for each provider')}
        </p>
      </div>
      <div className='mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-4'>
        {PROVIDER_RECHARGE_LINKS.map((provider) => (
          <a
            key={provider.name}
            href={provider.url}
            target='_blank'
            rel='noreferrer'
            className='bg-background/60 hover:bg-muted/60 flex items-center justify-between gap-2 rounded-xl border px-3 py-2.5 text-sm font-medium transition-colors'
          >
            <span className='truncate'>{provider.name}</span>
            <ExternalLink
              className='text-muted-foreground size-3.5 shrink-0'
              aria-hidden='true'
            />
          </a>
        ))}
      </div>
    </div>
  )
}
