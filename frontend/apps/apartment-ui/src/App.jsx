import React, { useState, useEffect, useRef } from 'react';
import PropTypes from 'prop-types';
import { createStore } from 'redux';
import { Provider } from 'react-redux';
import {
  ReactPlanner,
  reducer as ReactPlannerReducer,
  ReactPlannerActions,
  Catalog
} from 'react-planner';

// Обертка для старого контекста
class LegacyStoreProvider extends React.Component {
  getChildContext() { return { store: this.props.store }; }
  render() { return this.props.children; }
}
LegacyStoreProvider.childContextTypes = { store: PropTypes.object.isRequired };

// ==========================================
// 1. КАТАЛОГ ЭЛЕМЕНТОВ (НЕКЛИКАБЕЛЬНЫЕ)
// ==========================================
const MyCatalog = new Catalog();
const disableEvents = { pointerEvents: "none" }; // Стиль для некликабельности

MyCatalog.registerElement({
  name: 'wall', prototype: 'lines',
  info: { title: 'Wall', description: 'Wall', tag: ['wall'], image: '' },
  properties: { thickness: { type: 'length-measure', defaultValue: { length: 200 } } },
  render2D: (element, layer, scene) => {
    const thickness = element.getIn(['properties', 'thickness', 'length']) || 200;
    const v0Id = element.get('vertices').get(0); const v1Id = element.get('vertices').get(1);
    const v0 = layer.get('vertices').get(v0Id); const v1 = layer.get('vertices').get(v1Id);
    const distance = Math.sqrt(Math.pow(v0.get('x') - v1.get('x'), 2) + Math.pow(v0.get('y') - v1.get('y'), 2));
    const patternId = `hatch-${element.get('id')}`;
    return (
      <g style={disableEvents}>
        <defs><pattern id={patternId} width="40" height="40" patternTransform="rotate(45 0 0)" patternUnits="userSpaceOnUse"><line x1="0" y1="0" x2="0" y2="40" stroke="#aab7c4" strokeWidth="4" /></pattern></defs>
        <rect x="0" y={-thickness / 2} width={distance} height={thickness} fill="#ffffff" />
        <rect x="0" y={-thickness / 2} width={distance} height={thickness} fill={`url(#${patternId})`} stroke="#8395a7" strokeWidth="4" />
      </g>
    );
  },
  render3D: () => null 
});

MyCatalog.registerElement({
  name: 'door', prototype: 'holes',
  info: { title: 'Door', description: 'Door', tag: ['door'], image: '' },
  properties: { width: { type: 'length-measure', defaultValue: { length: 900 } } },
  render2D: (element, layer, scene) => {
    const width = element.getIn(['properties', 'width', 'length']) || 900;
    return <rect style={disableEvents} x={-width/2} y="-40" width={width} height="80" fill="#8e44ad" />;
  },
  render3D: () => null 
});

MyCatalog.registerElement({
  name: 'window', prototype: 'holes',
  info: { title: 'Window', description: 'Window', tag: ['window'], image: '' },
  properties: { width: { type: 'length-measure', defaultValue: { length: 1200 } } },
  render2D: (element, layer, scene) => {
    const width = element.getIn(['properties', 'width', 'length']) || 1200;
    return (
      <g style={disableEvents}>
        <rect x={-width/2} y="-50" width={width} height="100" fill="#ecf0f1" />
        <line x1={-width/2} y1="-15" x2={width/2} y2="-15" stroke="#3498db" strokeWidth="10" /><line x1={-width/2} y1="15" x2={width/2} y2="15" stroke="#3498db" strokeWidth="10" />
      </g>
    );
  },
  render3D: () => null 
});

MyCatalog.registerElement({
  name: 'area', prototype: 'areas',
  info: { title: 'Area', description: 'Room Area', tag: ['area'], image: '' },
  properties: {},
  render2D: (element, layer, scene) => {
    let path = "";
    element.get('vertices').forEach((vertexID, index) => {
      const vertex = layer.getIn(['vertices', vertexID]); path += (index === 0 ? "M" : "L") + `${vertex.get('x')} ${vertex.get('y')} `;
    });
    return <path style={disableEvents} d={path + "Z"} fill="rgba(41, 128, 185, 0.1)" stroke="none" />;
  },
  render3D: () => null 
});

// ==========================================
// 2. ДАННЫЕ И ПАРСЕР
// ==========================================
const store = createStore(ReactPlannerReducer);

const myFloorJson = {
  walls: [{id:"w1",points:[[0,0],[6000,0],[6000,5000],[0,5000],[0,0]],width:200}],
  doors: [{id:"d1",point:[1000,0],width:900},{id:"d2",point:[0,4000],width:700}],
  windows: [{id:"win1",points:[[4200,200],[5600,200]],height:1200}],
  rooms: [{id:"r_living",name:"Kitchen",area:[[0,0],[6000,0],[6000,5000],[0,5000]],area_m2:30}]
};

// --- КОНСТАНТЫ ГРАНИЦ ---
// Размеры комнаты (без отступов)
const ROOM_WIDTH = 6000;
const ROOM_HEIGHT = 5000;
// Отступ внутри виртуального холста, чтобы комната не жалась к краю 0,0
const ROOM_OFFSET_X = 500;
const ROOM_OFFSET_Y = 500;
// Виртуальный размер белого листа (комната + отступы)
const CANVAS_WIDTH = ROOM_WIDTH + ROOM_OFFSET_X * 2;
const CANVAS_HEIGHT = ROOM_HEIGHT + ROOM_OFFSET_Y * 2;
// Фиксированный масштаб (1 пиксель = 2.5 мм)
const VIEWER_SCALE = 0.4; 

function mapToPlannerState(customJson) {
  const vertices = {}; const lines = {}; const areas = {}; const holes = {};
  let vIdCounter = 0; let lIdCounter = 0;
  
  const roomVIds = [];
  customJson.walls.forEach((wall) => {
    const pts = wall.points;
    for (let i = 0; pts.length - 1 > i; i++) {
      const id = `v_${vIdCounter++}`;
      // ВНИМАНИЕ: Считаем координаты уже с учетом ROOM_OFFSET_X/Y
      vertices[id] = { id, type: "", prototype: "vertices", name: "Vertex", x: pts[i][0] + ROOM_OFFSET_X, y: pts[i][1] + ROOM_OFFSET_Y, lines: [], selected: false, misc: {} };
      roomVIds.push(id);
    }
    for (let i = 0; roomVIds.length > i; i++) {
      const id = `l_${lIdCounter++}`;
      const v1 = roomVIds[i]; const v2 = roomVIds[(i + 1) % roomVIds.length]; 
      lines[id] = { id, type: "wall", prototype: "lines", name: "Wall", properties: { height: { length: 3000 }, thickness: { length: wall.width } }, vertices: [v1, v2], holes: [], selected: false, misc: {} };
      vertices[v1].lines.push(id); vertices[v2].lines.push(id);
    }
  });

  if (customJson.rooms && customJson.rooms.length > 0) {
    areas["a_1"] = { id: "a_1", type: "area", prototype: "areas", name: customJson.rooms[0].name, properties: { patternColor: '#f1f2f6', thickness: 0 }, vertices: roomVIds, holes: [], selected: false, misc: {} };
  }

  const addHoleToClosestWall = (id, type, cx, cy, widthLength, heightLength) => {
    let bestLineId = null; let bestOffset = 0; let minDistance = Infinity;
    for (const lId in lines) {
      const line = lines[lId]; const v1 = vertices[line.vertices[0]]; const v2 = vertices[line.vertices[1]];
      // Точки проема тоже сдвигаем на ROOM_OFFSET_X/Y
      const hx = cx + ROOM_OFFSET_X; const hy = cy + ROOM_OFFSET_Y;
      const l2 = Math.pow(v1.x - v2.x, 2) + Math.pow(v1.y - v2.y, 2); if (l2 === 0) continue;
      let t = ((hx - v1.x) * (v2.x - v1.x) + (hy - v1.y) * (v2.y - v1.y)) / l2;
      t = Math.max(0, Math.min(1, t)); const projX = v1.x + t * (v2.x - v1.x); const projY = v1.y + t * (v2.y - v1.y); const dist = Math.sqrt(Math.pow(hx - projX, 2) + Math.pow(hy - projY, 2));
      if (dist < minDistance) { minDistance = dist; bestLineId = lId; bestOffset = t; }
    }
    if (bestLineId) { holes[id] = { id: id, type: type, prototype: "holes", name: type === 'door' ? "Door" : "Window", offset: bestOffset, line: bestLineId, properties: { width: { length: widthLength }, height: { length: heightLength }, altitude: { length: type === 'window' ? 800 : 0 } }, selected: false, misc: {} }; lines[bestLineId].holes.push(id); }
  };

  if (customJson.doors) customJson.doors.forEach(door => addHoleToClosestWall(door.id, 'door', door.point[0], door.point[1], door.width, 2000));
  if (customJson.windows) customJson.windows.forEach(win => { const [x1, y1] = win.points[0]; const [x2, y2] = win.points[1]; addHoleToClosestWall(win.id, 'window', (x1 + x2) / 2, (y1 + y2) / 2, Math.sqrt(Math.pow(x2 - x1, 2) + Math.pow(y2 - y1, 2)), win.height || 1200); });

  return { unit: "mm", layers: { "layer-1": { id: "layer-1", name: "Квартира", altitude: 0, order: 0, vertices, lines, holes, areas, items: {}, visible: true, opacity: 1 } }, grids: {}, guides: { horizontal: {}, vertical: {}, circular: {} }, 
    // Виртуальный белый лист строго по размеру комнаты с отступами
    width: CANVAS_WIDTH, height: CANVAS_HEIGHT 
  };
}

// ==========================================
// 3. КОМПОНЕНТ APP
// ==========================================
export default function App() {
  const [isLoaded, setIsLoaded] = useState(false);
  const containerRef = useRef(null); // Ссылка на div-обертку
  
  // Состояние нашего собственного перетаскивания (panning)
  const [panState, setPanState] = useState({
    active: false,
    startX: 0, startY: 0, // Где была мышка в момент клика
    translateX: 0, translateY: 0, // Текущее смещение плана в пикселях
  });

  useEffect(() => {
    const sceneData = mapToPlannerState(myFloorJson);
    store.dispatch(ReactPlannerActions.projectActions.loadProject(sceneData));
    
    // МЫ БОЛЬШЕ НЕ ДЕЛАЕМ ЗУМ В REDUX! Она сбрасывает его.
    setIsLoaded(true);
  }, []);

  // --- ОБРАБОТЧИКИ "ЖЕЛЕЗНОГО" ПЕРЕТАСКИВАНИЯ ---
  const handleMouseDown = (e) => {
    if (e.button !== 0) return; // Только левая кнопка
    setPanState({
      ...panState,
      active: true,
      startX: e.clientX - panState.translateX,
      startY: e.clientY - panState.translateY,
    });
    containerRef.current.style.cursor = 'grabbing';
  };

  const handleMouseMove = (e) => {
    if (!panState.active) return;
    e.preventDefault();

    // 1. Вычисляем желаемые координаты
    let newTx = e.clientX - panState.startX;
    let newTy = e.clientY - panState.startY;

    // 2. ЖЕСТКАЯ МАТЕМАТИКА ГРАНИЦ
    const rect = containerRef.current.getBoundingClientRect();
    
    // Масштабированные размеры комнаты на экране (в пикселях)
    const scaledWidth = CANVAS_WIDTH * VIEWER_SCALE;
    const scaledHeight = CANVAS_HEIGHT * VIEWER_SCALE;

    // -- Границы по оси X --
    // Ограничиваем так, чтобы левый край комнаты не улетел правее 0
    newTx = Math.min(0, newTx);
    // Ограничиваем так, чтобы правый край комнаты не улетел левее края экрана
    newTx = Math.max(rect.width - scaledWidth, newTx);

    // -- Границы по оси Y --
    // Ограничиваем так, чтобы верхний край комнаты не улетел ниже 0
    newTy = Math.min(0, newTy);
    // Ограничиваем так, чтобы нижний край комнаты не улетел выше края экрана
    newTy = Math.max(rect.height - scaledHeight, newTy);

    // 3. Применяем координаты только внутри разрешенного диапазона
    setPanState({
      ...panState,
      translateX: newTx,
      translateY: newTy,
    });
  };

  const handleMouseUp = () => {
    setPanState({ ...panState, active: false });
    containerRef.current.style.cursor = 'grab';
  };

  // Высчитываем ширину планировщика в пикселях (80% экрана)
  const plannerWidth = window.innerWidth * 0.8;

  if (!isLoaded) return <div>Загрузка...</div>;

  return (
    <div className="main-layout">
      {/* ЛЕВАЯ КОЛОНКА: Контролируемый План */}
      <div 
        className="plan-container" 
        ref={containerRef}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp} // Чтобы сбросить таск, если мышка улетела
      >
        {/* CSS-трансформер: Это div, который мы реально двигаем и зумим */}
        <div 
          className="plan-transformer"
          style={{
            transform: `translate(${panState.translateX}px, ${panState.translateY}px) scale(${VIEWER_SCALE})`
          }}
        >
          {/* Планировщик внутри. У него теперь pointer-events: none на стенах,
              так что клик будет ловиться контейнером-матрешкой. */}
          <Provider store={store}>
            <LegacyStoreProvider store={store}>
              <ReactPlanner 
                catalog={MyCatalog}
                width={CANVAS_WIDTH} // Режим 1:1, планировщик рисует всё
                height={CANVAS_HEIGHT}
                stateExtractor={(state) => state}
              />
            </LegacyStoreProvider>
          </Provider>
        </div>
      </div>

      {/* ПРАВАЯ КОЛОНКА: Твои карточки умных устройств */}
      <div className="device-panel">
        <h2>Smart Home</h2>
        <p>Квартира {ROOM_WIDTH/1000}м х {ROOM_HEIGHT/1000}м</p>
        
        <div className="device-card">
          <h4>Smart Lamp (Win1)</h4>
          <p>Включена | Яркость 80%</p>
          <button style={{pointerEvents: 'auto'}}>Выключить</button>
        </div>
        
        <div className="device-card">
          <h4>Temperature Sensor</h4>
          <p>23.5 °C | 55% Hum.</p>
        </div>
        
        <p style={{marginTop: 'auto', fontSize: '12px', opacity: 0.5}}>
          Ты сможешь сделать карточки кликабельными,<br/>так как они лежат вне SVG-планировщика.
        </p>
      </div>
    </div>
  );
}