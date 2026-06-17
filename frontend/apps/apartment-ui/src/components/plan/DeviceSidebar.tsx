import type { SmartDevice, Zone } from '../../types';
import { getZoneRoomId } from '../../utils/zones';
import { DEVICE_PATHS } from './devicePaths';

interface DeviceSidebarProps {
  devices: SmartDevice[];
  selectedZone: Zone | null;
  selectedDeviceId: string | null;
  onSelectDevice: (id: string | null) => void;
}

const DEVICE_NAMES: Record<SmartDevice['type'], string> = {
  smart_lamp: 'Умная лампа',
  smart_plug: 'Умная розетка',
  motion_sensor: 'Датчик движения',
  temperature_sensor: 'Датчик температуры',
  water_leak_sensor: 'Датчик протечки',
};

const sidebarTextStyle = {
  fontFamily: 'sans-serif',
} as const;

export function DeviceSidebar({
  devices,
  selectedZone,
  selectedDeviceId,
  onSelectDevice,
}: DeviceSidebarProps) {
  const selectedZoneRoomId = selectedZone ? getZoneRoomId(selectedZone) : null;

  return (
    <aside
      style={{
        width: '320px',
        height: '100%',
        minHeight: 0,
        backgroundColor: '#ffffff',
        borderLeft: '2px solid #ecf0f1',
        display: 'flex',
        flexDirection: 'column',
        boxShadow: '-4px 0 15px rgba(0,0,0,0.05)',
        zIndex: 10,
        overflow: 'hidden',
      }}
    >
      <div style={{ padding: '20px', borderBottom: '1px solid #ecf0f1' }}>
        <h2
          style={{
            margin: 0,
            fontSize: '18px',
            color: '#2c3e50',
            ...sidebarTextStyle,
          }}
        >
          Умные устройства
        </h2>
      </div>

      <section
        style={{
          padding: '14px 15px',
          borderBottom: '1px solid #ecf0f1',
          flex: '0 0 auto',
        }}
      >
        <div
          style={{
            height: '3px',
            width: '48px',
            borderRadius: '999px',
            backgroundColor: selectedZone ? '#2ecc71' : '#dfe6e9',
            marginBottom: '12px',
          }}
        />
        <div
          style={{
            border: `2px solid ${selectedZone ? '#2ecc71' : '#ecf0f1'}`,
            borderRadius: '8px',
            padding: '12px',
            backgroundColor: selectedZone ? '#f8fff9' : '#ffffff',
            ...sidebarTextStyle,
          }}
        >
          <div
            style={{
              fontSize: '14px',
              fontWeight: 700,
              color: selectedZone ? '#2c3e50' : '#95a5a6',
              marginBottom: selectedZone ? '10px' : 0,
            }}
          >
            Выбранная зона
          </div>
          {selectedZone && (
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: '6px',
                fontSize: '13px',
                color: '#34495e',
              }}
            >
              <div>
                <strong style={{ color: '#95a5a6' }}>ID:</strong> {selectedZone.id}
              </div>
              <div>
                <strong style={{ color: '#95a5a6' }}>Комната:</strong>{' '}
                {selectedZoneRoomId ?? 'не указана'}
              </div>
            </div>
          )}
        </div>
      </section>

      <div
        style={{
          padding: '15px',
          display: 'flex',
          flexDirection: 'column',
          gap: '10px',
          flex: '1 1 auto',
          minHeight: 0,
          overflowY: 'auto',
        }}
      >
        {devices.map((device) => {
          const isSelected = selectedDeviceId === device.id;
          const color = isSelected ? '#2ecc71' : '#bdc3c7';

          return (
            <div
              key={device.id}
              onClick={() => onSelectDevice(device.id)}
              style={{
                border: `2px solid ${isSelected ? '#2ecc71' : '#ecf0f1'}`,
                borderRadius: '8px',
                padding: '12px',
                cursor: 'pointer',
                transition: 'all 0.2s',
                backgroundColor: isSelected ? '#f8fff9' : '#ffffff',
                flex: '0 0 auto',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <svg width="24" height="24" viewBox="0 0 24 24" fill={color}>
                  <path d={DEVICE_PATHS[device.type]} />
                </svg>
                <span
                  style={{
                    fontWeight: isSelected ? 'bold' : 'normal',
                    color: isSelected ? '#2c3e50' : '#7f8c8d',
                    fontSize: '15px',
                    ...sidebarTextStyle,
                  }}
                >
                  {DEVICE_NAMES[device.type]}
                </span>
              </div>

              {isSelected && (
                <div
                  style={{
                    marginTop: '15px',
                    paddingTop: '15px',
                    borderTop: '1px solid #e0e6ed',
                    fontSize: '13px',
                    color: '#34495e',
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '6px',
                    ...sidebarTextStyle,
                  }}
                >
                  <div>
                    <strong style={{ color: '#95a5a6' }}>ID:</strong> {device.id}
                  </div>
                  <div>
                    <strong style={{ color: '#95a5a6' }}>Комната:</strong>{' '}
                    {device.room_id}
                  </div>
                  <div>
                    <strong style={{ color: '#95a5a6' }}>Статус:</strong> Онлайн
                  </div>
                  <div>
                    <strong style={{ color: '#95a5a6' }}>Заряд:</strong> 85%
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </aside>
  );
}
