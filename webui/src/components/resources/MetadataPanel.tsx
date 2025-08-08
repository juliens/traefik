import { FiInfo } from 'react-icons/fi'

import { Chips, DetailSection, ItemBlock } from './DetailSections'

type MetadataPanelProps = {
  metadata?: Record<string, any>
}

const MetadataPanel = ({ metadata }: MetadataPanelProps) => {
  if (!metadata || Object.keys(metadata).length === 0) {
    return null
  }

  const formatMetadataValue = (value: any): string => {
    if (typeof value === 'string') {
      return value
    }
    if (typeof value === 'number' || typeof value === 'boolean') {
      return String(value)
    }
    if (typeof value === 'object' && value !== null) {
      return JSON.stringify(value, null, 2)
    }
    return String(value)
  }

  const metadataItems = Object.entries(metadata).map(([key, value]) => `${key}: ${formatMetadataValue(value)}`)

  return (
    <DetailSection icon={<FiInfo size={20} />} title="Metadata">
      <ItemBlock title="Properties">
        <Chips items={metadataItems} variant="neon" />
      </ItemBlock>
    </DetailSection>
  )
}

export default MetadataPanel