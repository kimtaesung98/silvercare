import 'dart:math' as math;

import 'package:flutter/material.dart';

import 'elder_controller.dart';
import 'tokens.dart';

/// 피그마 컴포넌트 `표시/마이크`. 가운데 원(240)과 바깥 물결 두 겹(전체 420).
/// 듣는 중과 말하는 중에는 물결이 숨 쉬듯 커졌다 작아집니다.
class MicIndicator extends StatefulWidget {
  const MicIndicator({super.key, required this.state});

  final MicState state;

  static const size = 420.0;
  static const core = 240.0;

  @override
  State<MicIndicator> createState() => _MicIndicatorState();
}

class _MicIndicatorState extends State<MicIndicator> with SingleTickerProviderStateMixin {
  late final _breath = AnimationController(vsync: this, duration: const Duration(milliseconds: 1600));

  @override
  void initState() {
    super.initState();
    _sync();
  }

  @override
  void didUpdateWidget(MicIndicator old) {
    super.didUpdateWidget(old);
    _sync();
  }

  void _sync() {
    if (widget.state == MicState.idle) {
      _breath.stop();
      _breath.value = 0;
    } else if (!_breath.isAnimating) {
      _breath.repeat(reverse: true);
    }
  }

  @override
  void dispose() {
    _breath.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final color = switch (widget.state) {
      MicState.idle => Tokens.textDim,
      MicState.listening => Tokens.listening,
      MicState.speaking => Tokens.accent,
    };
    final idle = widget.state == MicState.idle;
    return SizedBox.square(
      dimension: MicIndicator.size,
      child: AnimatedBuilder(
        animation: _breath,
        builder: (context, _) {
          final t = Curves.easeInOut.transform(_breath.value);
          return Stack(
            alignment: Alignment.center,
            children: [
              if (!idle) ...[
                _circle(MicIndicator.size * (0.88 + 0.12 * t), color.withValues(alpha: 0.12)),
                _circle(MicIndicator.size * (0.70 + 0.08 * t), color.withValues(alpha: 0.22)),
              ],
              _circle(
                MicIndicator.core,
                idle ? Tokens.textDim.withValues(alpha: 0.18) : color,
                child: widget.state == MicState.speaking
                    ? _Bars(t: _breath.value)
                    : Icon(Icons.mic_none_rounded, size: 110, color: idle ? Tokens.textDim : Tokens.white),
              ),
            ],
          );
        },
      ),
    );
  }

  Widget _circle(double d, Color c, {Widget? child}) => Container(
    key: child == null ? null : const Key('mic-core'),
    width: d,
    height: d,
    decoration: BoxDecoration(color: c, shape: BoxShape.circle),
    alignment: Alignment.center,
    child: child,
  );
}

/// 말하는 중의 소리 막대 다섯 개.
class _Bars extends StatelessWidget {
  const _Bars({required this.t});

  final double t;

  @override
  Widget build(BuildContext context) {
    const base = [0.35, 0.65, 1.0, 0.65, 0.35];
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        for (var i = 0; i < base.length; i++)
          Container(
            margin: const EdgeInsets.symmetric(horizontal: 9),
            width: 16,
            height: 120 * (base[i] * (0.7 + 0.3 * math.sin(math.pi * (t + i * 0.2)).abs())),
            decoration: BoxDecoration(color: Tokens.white, borderRadius: BorderRadius.circular(8)),
          ),
      ],
    );
  }
}
