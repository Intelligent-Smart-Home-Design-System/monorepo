import React from 'react';
import type { SmartDevice } from '../../types';
import { DEVICE_PATHS } from './SmartDevices';

interface DeviceSidebarProps {
  devices: SmartDevice[];
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

export function DeviceSidebar({ devices, selectedDeviceId, onSelectDevice }: DeviceSidebarProps) {
  return (
    <div style={{
      width: '320px',
      height: '100vh',
      backgroundColor: '#ffffff',
      borderLeft: '2px solid #ecf0f1',
      display: 'flex',
      flexDirection: 'column',
      boxShadow: '-4px 0 15px rgba(0,0,0,0.05)',
      zIndex: 10,
      overflowY: 'auto'
    }}>
      <div style={{ padding: '20px', borderBottom: '1px solid #ecf0f1' }}>
        <h2 style={{ margin: 0, fontSize: '18px', color: '#2c3e50', fontFamily: 'sans-serif' }}>
          Умные устройства
        </h2>
      </div>

      <div style={{ padding: '15px', display: 'flex', flexDirection: 'column', gap: '10px' }}>
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
                backgroundColor: isSelected ? '#f8fff9' : '#ffffff'
              }}
            >
              {/* Шапка карточки (Иконка + Название) */}
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                {/* Рендерим SVG иконку прямо в HTML */}
                <svg width="24" height="24" viewBox="0 0 24 24" fill={color}>
                  <path d={DEVICE_PATHS[device.type]} />
                </svg>
                <span style={{ 
                  fontWeight: isSelected ? 'bold' : 'normal',
                  color: isSelected ? '#2c3e50' : '#7f8c8d',
                  fontFamily: 'sans-serif',
                  fontSize: '15px'
                }}>
                  {DEVICE_NAMES[device.type]}
                </span>
              </div>

              {/* Развернутая часть с тестовыми данными (показывается только если выбрано) */}
              {isSelected && (
                <div style={{ 
                  marginTop: '15px', 
                  paddingTop: '15px', 
                  borderTop: '1px solid #e0e6ed',
                  fontSize: '13px',
                  color: '#34495e',
                  fontFamily: 'sans-serif',
                  display: 'flex',
                  flexDirection: 'column',
                  gap: '6px'
                }}>
                  <div><strong style={{ color: '#95a5a6' }}>ID:</strong> {device.id}</div>
                  <div><strong style={{ color: '#95a5a6' }}>Комната:</strong> {device.room_id}</div>
                  <div><strong style={{ color: '#95a5a6' }}>Статус:</strong> Онлайн</div>
                  <div><strong style={{ color: '#95a5a6' }}>Заряд:</strong> 85%</div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}