import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// 지도 바탕 타일. 개발 중에는 OpenStreetMap을 쓰고, 운영 전에 카카오·티맵 등
/// 상용 타일로 바꿉니다. 테스트는 null로 바꿔 네트워크를 쓰지 않습니다.
final mapTilesProvider = Provider<TileLayer?>(
  (ref) => TileLayer(
    urlTemplate: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
    userAgentPackageName: 'kr.notchiwon.caregiver',
  ),
);
