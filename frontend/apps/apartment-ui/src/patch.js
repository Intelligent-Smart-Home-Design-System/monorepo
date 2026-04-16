window.global = window; // Наш новый спасительный костыль для старого lodash

import React from 'react';
import PropTypes from 'prop-types';
import createReactClass from 'create-react-class';

// Возвращаем удаленные методы обратно в ядро React, 
// чтобы старые библиотеки не падали с ошибками
React.PropTypes = PropTypes;
React.createClass = createReactClass;